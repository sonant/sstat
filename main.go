package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/sirupsen/logrus"
)

var (
	logger   = logrus.New()
	endpoint string
	db       gorm.DB
)

//SystemData sturcture
type SystemData struct {
	gorm.Model
	CPULoad float32
	MEMFree int32
}

func init() {
	logger.SetFormatter(&logrus.JSONFormatter{})
}

func collectData(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ctx.Done():
			logger.Infoln("Exit from collect data....")
			ticker.Stop()
			wg.Done()
			return
		case <-ticker.C:
			logger.Infoln("I'm collecting data for you now...")
		}
	}
}

func sendData(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(time.Second * 10)
	for {
		select {
		case <-ctx.Done():
			logger.Infoln("Exit from send data....")
			ticker.Stop()
			wg.Done()
			return
		case <-ticker.C:
			logger.Infoln("I'm sending data to endpoin now...")
		}
	}
}

func main() {
	exit := make(chan os.Signal, 1)
	var wg sync.WaitGroup
	flag.StringVar(&endpoint, "endpoint", "", "Address of remote server, which recive logs.")
	flag.Parse()

	//prepare database connection
	db, err := gorm.Open("sqlite3", "./db.db")
	if err != nil {
		logger.Errorln(err)
	}
	defer db.Close()

	db.AutoMigrate(&SystemData{})
	ctx, cancel := context.WithCancel(context.Background())

	signal.Notify(exit, os.Interrupt)
	wg.Add(1)
	go collectData(ctx, &wg)
	wg.Add(1)
	go sendData(ctx, &wg)

	<-exit

	logger.Infoln("Got exit signal. Stoping...")
	//cancel collectionData() executing
	cancel()
	//wait until collectData() do everything
	wg.Wait()
	//say BYE! everyONE
	logger.Infoln("BYE!")
	os.Exit(0)

}
