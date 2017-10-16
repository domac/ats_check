package util

import (
	"github.com/valyala/fasthttp"
	"time"
)

func NewFastHttpHostClient(proxy string, readTimeout time.Duration) *fasthttp.HostClient {
	c := &fasthttp.HostClient{
		Addr:        proxy,
		ReadTimeout: readTimeout,
		Name:        "request",
	}
	return c
}

func NewFastHttpClient(readTimeout time.Duration) *fasthttp.Client {
	c := &fasthttp.Client{
		ReadTimeout: readTimeout,
		Name:        "ats-parent-check",
	}
	return c
}
