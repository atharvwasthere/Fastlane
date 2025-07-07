package client

import (
	"bufio"
	"net"
	"time"
)

type Client struct {
	conn net.Conn
	reader *bufio.Reader
}

func NewClient() *Client{
	return &Client{}
}

func (c * Client) Connect(host string) error {
	conn, err := net.Dial("tcp",host+":8080") // tcp port used 
	if err != nil {
		return err
	}
	c.conn = conn
	c.reader = bufio.NewReader(conn)
	return nil
}

func (c *Client) Ping() (time.Duration , error){
	start := time.Now()
	_, err := c.conn.Write([]byte("PING\n"))
	if err != nil {
		return 0, err
	}
	_, err = c.reader.ReadBytes('\n');
	return time.Since(start), err
}