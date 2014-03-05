package main

import (
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	UserAgent string
	client    *http.Client
	EndPoint  *url.URL
}

func NewClient(endpoint *url.URL) *Client {
	return &Client{
		UserAgent: "Mezasi/0.1",
		client: &http.Client{
			Transport: &http.Transport{
				Dial: func(netw, addr string) (net.Conn, error) {
					return net.DialTimeout(netw, addr, time.Duration(time.Second*300))
				},
			},
		},
		EndPoint: endpoint,
	}
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.client.Do(req)
}

func (c *Client) NewRequest(method string, pathStr string, body io.Reader) (*http.Request, error) {
	urlPath, err := url.Parse(pathStr)
	if err != nil {
		return nil, err
	}
	u := c.EndPoint.ResolveReference(urlPath)
	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", c.UserAgent)
	return req, nil
}
