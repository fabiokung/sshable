package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
)

type Rendezvous struct {
	Address *url.URL
}

func NewRendezvous(addr string) (*Rendezvous, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	return &Rendezvous{u}, nil
}

func (r *Rendezvous) Connect() error {
	conn, err := r.rendezvousConn()
	if err != nil  {
		return err
	}
	defer conn.Close()

	sshConn, err := r.sshConn()
	if err != nil {
		return err
	}
	defer sshConn.Close()

	inDone := make(chan bool)
	outDone := make(chan bool)
	go forward(conn, sshConn, inDone)
	go forward(sshConn, conn, outDone)

	<-inDone
	<-outDone
	return nil
}

func forward(reader net.Conn, writer net.Conn, done chan bool) error {
	defer func() { done <- true }()

	buf := make([]byte, 2048)
	for {
		n, err := reader.Read(buf)
		if err != nil {
			return err
		}
		if _, err = writer.Write(buf[:n]); err != nil {
			return err
		}
	}
	return nil
}

func (r *Rendezvous) rendezvousConn() (net.Conn, error) {
	conn, err := tls.Dial("tcp", r.Address.Host, nil)
	if err != nil {
		return nil, err
	}
	if _, err = fmt.Fprintln(conn, r.Address.Path); err != nil {
		conn.Close()
		return nil, err
	}
	if _, err = bufio.NewReader(conn).ReadString('\n'); err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

func (r *Rendezvous) sshConn() (net.Conn, error) {
	return net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", SSHD_PORT))
}
