package etcdclient

import (
	"net/url"
	"strings"
	"time"

	"github.com/coreos/etcd/client"
	"golang.org/x/net/context"
)

type EClient struct {
	client client.Client
	kapi   client.KeysAPI
}

func NewEClient(host string) *EClient {
	machines := strings.Split(host, ",")
	initEtcdPeers(machines)
	cfg := client.Config{
		Endpoints:               machines,
		Transport:               client.DefaultTransport,
		HeaderTimeoutPerRequest: time.Second,
	}

	c, err := client.New(cfg)
	if err != nil {
		return nil
	}

	return &EClient{
		client: c,
		kapi:   client.NewKeysAPI(c),
	}
}

func (self *EClient) Get(key string, sort, recursive bool) (*client.Response, error) {
	getOptions := &client.GetOptions{
		Recursive: recursive,
		Sort:      sort,
	}
	return self.kapi.Get(context.Background(), key, getOptions)
}

func (self *EClient) Create(key string, value string, ttl uint64) (*client.Response, error) {
	return self.kapi.Create(context.Background(), key, value)
}

func (self *EClient) Delete(key string, recursive bool, dir bool) (*client.Response, error) {
	delOptions := &client.DeleteOptions{
		Recursive: recursive,
		Dir:       dir,
	}
	return self.kapi.Delete(context.Background(), key, delOptions)
}

func (self *EClient) CreateDir(key string, ttl uint64) (*client.Response, error) {
	setOptions := &client.SetOptions{
		TTL: time.Duration(ttl) * time.Second,
		Dir: true,
	}
	return self.kapi.Set(context.Background(), key, "", setOptions)
}

func (self *EClient) Set(key string, value string, ttl uint64) (*client.Response, error) {
	setOptions := &client.SetOptions{
		TTL: time.Duration(ttl) * time.Second,
	}
	return self.kapi.Set(context.Background(), key, value, setOptions)
}

func (self *EClient) SetWithTTL(key string, ttl uint64) (*client.Response, error) {
	setOptions := &client.SetOptions{
		TTL:       time.Duration(ttl) * time.Second,
		Refresh:   true,
		PrevExist: client.PrevExist,
	}
	return self.kapi.Set(context.Background(), key, "", setOptions)
}

func (self *EClient) CompareAndSwap(key string, value string, ttl uint64, prevValue string, prevIndex uint64) (*client.Response, error) {
	refresh := false
	if ttl > 0 {
		refresh = true
	}
	setOptions := &client.SetOptions{
		PrevValue: prevValue,
		PrevIndex: prevIndex,
		TTL:       time.Duration(ttl) * time.Second,
		Refresh:   refresh,
	}
	return self.kapi.Set(context.Background(), key, value, setOptions)
}

func (self *EClient) Watch(key string, waitIndex uint64, recursive bool) client.Watcher {
	watchOptions := &client.WatcherOptions{
		AfterIndex: waitIndex,
		Recursive:  recursive,
	}
	return self.kapi.Watcher(key, watchOptions)
}

func initEtcdPeers(machines []string) error {
	for i, ep := range machines {
		u, err := url.Parse(ep)
		if err != nil {
			return err
		}
		if u.Scheme == "" {
			u.Scheme = "http"
		}
		machines[i] = u.String()
	}
	return nil
}
