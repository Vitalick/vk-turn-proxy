package main

var userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:144.0) Gecko/20100101 Firefox/144.0"

type getCredsFunc func(string) (string, string, string, error)
