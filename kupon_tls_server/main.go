package main

import (
	"context"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/google/uuid"
	"io"
	"net"
	"nexusws/cmd/kupon_tls_server/sliprecord"
	"nexusws/cmd/kupon_tls_server/slipvalidation"
	"nexusws/cmd/kupon_tls_server/zreport"
	nxCtx "nexusws/pkg/context"
	"nexusws/pkg/httpclient"
	"nexusws/pkg/nexushttpclient"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {

	cfg := &Config{}
	err := NewFromFile("./config.yaml", cfg)
	if err != nil {
		panic(err)
	}

	f, err := os.OpenFile("log.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	var logger log.Logger
	{
		logger = log.NewLogfmtLogger(f /*os.Stderr*/)
		logger = log.NewSyncLogger(logger)
		logger = log.With(logger,
			"service", "NexusTlsServer", "time:", log.DefaultTimestampUTC, "caller", log.DefaultCaller)
	}
	level.Info(logger).Log("msg", "kupon service started")
	defer level.Info(logger).Log("msg", "kupon service stopped")

	cer, err := tls.LoadX509KeyPair("kuponServerCertificate/server.pem", "kuponServerCertificate/server.key")
	if err != nil {
		level.Error(logger).Log("err", err)
		return
	}

	tlsServerListen := fmt.Sprintf("%s:%s", cfg.TLSServer.ListenIP, cfg.TLSServer.ListenPort)

	config := &tls.Config{Certificates: []tls.Certificate{cer},
		MaxVersion: tls.VersionTLS12,
		MinVersion: tls.VersionTLS12,
		ClientAuth: tls.NoClientCert,
		//
		//VerifyConnection: func(cs tls.ConnectionState) error {
		//	opts := x509.VerifyOptions{
		//		DNSName:       cs.ServerName,
		//		Intermediates: x509.NewCertPool(),
		//		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		//	}
		//	for _, cert := range cs.PeerCertificates[1:] {
		//		opts.Intermediates.AddCert(cert)
		//	}
		//	_, err := cs.PeerCertificates[0].Verify(opts)
		//	level.Error(logger).Log("client_certificate", err)
		//	return nil
		//},
		//VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
		//	return nil
		//},
		GetConfigForClient: func(info *tls.ClientHelloInfo) (*tls.Config, error) {

			//s := fmt.Sprintf("+%v", info.CipherSuites)
			//level.Error(logger).Log("client_ciphersuite", s)
			//
			//s = fmt.Sprintf("+%v", info.SupportedProtos)
			//level.Error(logger).Log("client_protos", s)
			//
			//s = fmt.Sprintf("+%v", info)
			//level.Error(logger).Log("client_info", s)

			return nil, nil
		},
	}
	ln, err := tls.Listen("tcp", tlsServerListen, config)
	if err != nil {
		level.Error(logger).Log("err", err)
		return
	}
	defer ln.Close()

	errs := make(chan error)
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errs <- fmt.Errorf("%s", <-c)
		ln.Close()
	}()

	hcl := httpclient.NewHttpInternalServiceClient(logger)
	nCl := nexushttpclient.New(logger, hcl)

	nexusWsHost := fmt.Sprintf("%s:%d", cfg.NexusWS.Host, cfg.NexusWS.Port)
	nexusCl := nexushttpclient.NewSlipClient(&logger, nCl, nexusWsHost)

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				level.Error(logger).Log("err", err)
				continue
			}
			ctx := context.Background()

			traceID := uuid.New().String()

			ctx = nxCtx.WithTraceID(ctx, traceID)
			ctx = nxCtx.WithServiceName(ctx, cfg.Runtime)

			l := log.With(logger, "trace_id", nxCtx.GetTraceId(ctx))

			go handleConnection(ctx, conn, nexusCl, l)
		}
	}()
	level.Error(logger).Log("exit", <-errs)
}

func handleConnection(ctx context.Context, conn net.Conn, nexusCl *nexushttpclient.SlipClient, logger log.Logger) {
	defer func() {
		//level.Error(logger).Log("err", "closing socket")

		conn.Close()
	}()

	level.Info(logger).Log("newconnection", conn.RemoteAddr())

	// set SetReadDeadline
	err := conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
	if err != nil {
		level.Error(logger).Log("SetReadDeadline failed:", err)
		return
	}

	for {
		msg := make([]byte, sliprecord.SlipMaxMessageLength)
		n, err := conn.Read(msg)
		if err != nil {
			if err == io.EOF {
				//level.Info(logger).Log("info", "reached end of data from socket")
			} else if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// time out
				//level.Error(logger).Log("read timeout:", err)
				return

			} else {
				level.Error(logger).Log("err", err)
				return
			}
		}

		//make sure we have received message Identifier G or E
		if n < sliprecord.SlipRecordIdentifierLength {
			//level.Error(logger).Log("expected at least message identifier", hex.EncodeToString(msg))
			return
		}

		if msg[sliprecord.SlipRecordIdentifierOffset] == sliprecord.SlipRecordMessageIdentifier {
			level.Info(logger).Log("newmessage", "G")

			s := sliprecord.New(logger, nexusCl, msg, n)
			err = s.HandleMsgG(ctx, conn)

			//d := s.RawMessage()
			//level.Info(logger).Log("received raw message G:", hex.EncodeToString(d))
			if err != nil {
				level.Error(logger).Log("err", err)
			}
		} else if msg[sliprecord.SlipRecordIdentifierOffset] == sliprecord.SlipValidationMessageIdentifier {
			level.Info(logger).Log("newmessage", "E")

			s := slipvalidation.New(logger, nexusCl, msg, n)
			err = s.Handle(ctx, conn)

			//d, i := s.RawMessage()
			//level.Info(logger).Log("received raw message E:", hex.EncodeToString(d[:i]))
			if err != nil {
				level.Error(logger).Log("err", err)
			}
		} else if msg[sliprecord.SlipRecordIdentifierOffset] == zreport.ZReportMessageIdentifier {
			level.Info(logger).Log("newmessage", "W")

			s := zreport.New(logger, nexusCl, msg, n)
			err = s.Handle(ctx, conn)

			//d := s.RawMessage()
			//level.Info(logger).Log("received raw message W:", hex.EncodeToString(d))
			if err != nil {
				d := s.RawMessage()
				level.Info(logger).Log("received raw message W:", hex.EncodeToString(d))
				level.Error(logger).Log("err", err)
			}
		} else {
			level.Error(logger).Log("unknown message identifier", msg)
		}
	}
}
