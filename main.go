package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/shirou/gopsutil/mem"
	"github.com/sirupsen/logrus"
)

var (
	logger    = logrus.New()
	endpoint  string
	db        *gorm.DB
	transport = &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}
	client = &http.Client{
		Timeout:   time.Second * 10,
		Transport: transport,
	}
)

// SystemData sturcture.
type SystemData struct {
	gorm.Model
	CPULoad uint64
	MEMFree uint64
}

func init() {
	logger.SetFormatter(&logrus.JSONFormatter{})
}

// collectData collect data from system every 1 second.
func collectData(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(time.Second * 1)
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			wg.Done()
			return
		case <-ticker.C:
			m, _ := mem.VirtualMemory()
			db.Create(&SystemData{CPULoad: 10000, MEMFree: m.Free})
		}
	}
}

// sendData send data to endpoint every 10 seconds and clean DB if success.
func sendData(ctx context.Context, wg *sync.WaitGroup) {
	var data []SystemData
	ticker := time.NewTicker(time.Second * 10)
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			wg.Done()
			return
		case <-ticker.C:
			db.Find(&data)
			d, _ := json.Marshal(&data)
			// client.Post
			fmt.Printf("%s\n", d)
			db.Delete(&data)
			data = data[:0]
		}
	}
}

func main() {
	var err error
	exit := make(chan os.Signal, 1)
	var wg sync.WaitGroup
	flag.StringVar(&endpoint, "endpoint", "", "Address of remote server, which recive logs.")
	flag.Parse()

	// Prepare database connection.
	db, err = gorm.Open("sqlite3", "./db.db")
	if err != nil {
		logger.Errorln(err)
	}
	db.DB().SetMaxIdleConns(0)
	db.AutoMigrate(&SystemData{})
	ctx, cancel := context.WithCancel(context.Background())

	signal.Notify(exit, os.Interrupt)
	wg.Add(1)
	go collectData(ctx, &wg)
	wg.Add(1)
	go sendData(ctx, &wg)
	logger.Infoln("I'am working...")
	<-exit

	logger.Infoln("Got exit signal. Stoping...")
	// Cancel collectionData() executing.
	cancel()
	// Wait until collectData() do everything.
	wg.Wait()
	db.Close()

	// Say BYE! everyONE.
	logger.Infoln("BYE!")
	os.Exit(0)

}
