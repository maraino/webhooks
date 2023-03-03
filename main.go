package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/glebarez/go-sqlite"
	"github.com/smallstep/logging"
	"github.com/smallstep/logging/httplog"
	"go.step.sm/crypto/pemutil"
	"go.step.sm/crypto/tlsutil"

	"github.com/maraino/webhooks/pkg/server"
)

func main() {
	var address, database string
	var rootFile, certFile, keyFile string
	flag.StringVar(&address, "address", ":3000", "The TCP `address` to listen on (e.g. ':8443').")
	flag.StringVar(&database, "database", "db/database.sqlite3", "The TCP `address` to listen on (e.g. ':8443').")
	flag.StringVar(&rootFile, "root", "", "The `path` to the Root CA bundle to use.")
	flag.StringVar(&certFile, "cert", "", "The `path` to the certificate to use.")
	flag.StringVar(&keyFile, "key", "", "The `path` to the certificate key to use.")

	flag.Parse()

	switch {
	case address == "":
		fail("flag --address cannot be empty")
	case database == "":
		fail("flag --database cannot be empty")
	case certFile != "" && keyFile == "":
		fail("flag --cert requires the flag --key")
	case keyFile != "" && certFile == "":
		fail("flag --key requires the flag --cert")
	}

	var tlsConfig *tls.Config
	if certFile != "" && keyFile != "" {
		creds, err := newServerCredentialsFromFile(certFile, keyFile, rootFile)
		if err != nil {
			failErr(err)
		}
		tlsConfig = creds.TLSConfig()
	}

	logger, err := logging.New("webhooks", logging.WithFormatJSON())
	if err != nil {
		failErr(err)
	}

	db, err := sql.Open("sqlite", database)
	if err != nil {
		failErr(err)
	}
	if err := db.Ping(); err != nil {
		failErr(err)
	}

	srv := &http.Server{
		Addr:      address,
		Handler:   httplog.Middleware(logger, &server.Webhook{DB: db}),
		TLSConfig: tlsConfig,
		ErrorLog:  logger.StdLogger(logging.ErrorLevel),
	}

	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(signals)

		for sig := range signals {
			switch sig {
			case syscall.SIGINT, syscall.SIGTERM:
				log.Println("shutting down ...")
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				if err := srv.Shutdown(ctx); err != nil {
					log.Println(err)
				}
				cancel()
				if err := db.Close(); err != nil {
					log.Println(err)
				}
				return
			}
		}
	}()

	if srv.TLSConfig != nil {
		log.Println("starting https server at", address)
		if err := srv.ListenAndServeTLS("", ""); !errors.Is(err, http.ErrServerClosed) {
			log.Println(err)
		}
	} else {
		log.Println("starting http server at", address)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Println(err)
		}
	}
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format, args...)
	fmt.Fprintln(os.Stderr)
	os.Exit(1)
}

func failErr(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

// newServerCredentialsFromFile returns a ServerCredentials that renews the
// certificate from a file on disk.
func newServerCredentialsFromFile(certFile, keyFile, rootFile string) (*tlsutil.ServerCredentials, error) {
	if _, err := tls.LoadX509KeyPair(certFile, keyFile); err != nil {
		return nil, fmt.Errorf("error loading certificate: %w", err)
	}
	return tlsutil.NewServerCredentials(func(*tls.ClientHelloInfo) (*tls.Certificate, *tls.Config, error) {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, nil, fmt.Errorf("error loading certificate: %w", err)
		}
		if cert.Leaf == nil {
			cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
			if err != nil {
				return nil, nil, fmt.Errorf("error parsing certificate: %w", err)
			}
		}

		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
		if rootFile != "" {
			roots, err := pemutil.ReadCertificateBundle(rootFile)
			if err != nil {
				return nil, nil, err
			}
			tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
			tlsConfig.ClientCAs = x509.NewCertPool()
			for _, crt := range roots {
				tlsConfig.ClientCAs.AddCert(crt)
			}
		}

		return &cert, tlsConfig, nil
	})
}
