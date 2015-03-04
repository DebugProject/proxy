package main

import (
	"bytes"
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/golang/groupcache"
)

var (
	myAddress    string
	proxyAddress string
	cacheSize    int64
	pool         *groupcache.HTTPPool
)

func init() {
	flag.StringVar(&myAddress, "address", "127.0.0.1:8053", "HTTP Address where to bind")
	flag.StringVar(&proxyAddress, "proxy-address", "http://google.com", "HTTP Address where to bind")
	flag.Int64Var(&cacheSize, "cache-size", 2<<30, "Cache size in bytes")
}

func main() {
	flag.Parse()

	pool = groupcache.NewHTTPPool("http://" + myAddress)

	cache := groupcache.NewGroup("cache", cacheSize, groupcache.GetterFunc(
		func(ctx groupcache.Context, key string, dest groupcache.Sink) error {
			res, err := http.Get(proxyAddress + key)
			if err != nil {
				return err
			}

			defer res.Body.Close()

			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				return err
			}

			if res.StatusCode != 200 {
				return errors.New(string(body))
			}

			dest.SetBytes(body)
			return nil
		}))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var result []byte
		if err := cache.Get(nil, r.URL.RequestURI(), groupcache.AllocatingByteSliceSink(&result)); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		http.ServeContent(w, r, "ololo", time.Now(), bytes.NewReader(result))
	})

	log.Fatal(http.ListenAndServe(myAddress, nil))
}
