package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/kvu787/go-schedule/scraper/config"
	"github.com/kvu787/go-schedule/scraper/database"
	"github.com/kvu787/go-schedule/scraper/extract"
	"github.com/kvu787/go-schedule/scraper/fetch"
	_ "github.com/lib/pq"
)

func main() {
	if err := os.Chdir(os.ExpandEnv(config.AppRoot)); err != nil { // is this idiomatic and safe?
		fmt.Println(err)
		return
	}
	for {
		// open switch db
		switchDB, err := sql.Open(config.DbConnSwitch.Driver(), config.DbConnSwitch.Conn())
		if err != nil {
			log.Fatalln("Failed to open switch db")
			log.Fatalln(err)
			return
		}
		defer switchDB.Close()
		// determine which app db to use
		var db *sql.DB
		if db, err = database.GetAppDB(switchDB, true); err != nil {
			log.Fatalln("Failed to determine app db")
			log.Fatalln(err)
			return
		}
		defer db.Close()
		// run setup sql against db
		if err = database.SetupDB(db); err != nil {
			log.Fatalln("Failed to setup app db")
			log.Fatalln(err)
			return
		}
		// scrape
		client := http.DefaultClient
		if err = scrape(client, db, true); err != nil {
			log.Fatalln("Scraping failed")
			log.Fatalln(err)
			return
		}
		// flip switch
		if err = database.FlipSwitch(switchDB); err != nil {
			log.Fatalln("Failed to flip switch db")
			log.Fatalln(err)
			return
		}
		time.Sleep(config.ScraperTimeout)
	}
}

func scrape(c *http.Client, db *sql.DB, concurrent bool) error {
	deptIndex, err := fetch.Get(c, config.RootIndex)
	if err != nil {
		log.Fatalln("Failed to fetch RootIndex (department page)")
		return err
	}
	depts := extract.DeptIndex(deptIndex).Extract(nil)
	if concurrent {
		fmt.Println("Scraper started in concurrent mode")
		quitc := make(chan int)
		for _, dept := range depts {
			go func(dept database.Dept) {
				scrapeDept(dept, c, db)
				quitc <- 1
			}(dept)
			time.Sleep(config.ScraperFetchTimeout)
		}
		for i := 0; i < len(depts); i++ {
			<-quitc
		}
	} else {
		fmt.Println("Scraper started in non-concurrent mode")
		for _, dept := range depts {
			fetch.Get(c, dept.Link)
			scrapeDept(dept, c, db)
		}
	}
	fmt.Println("Scraper done")
	return nil
}

func scrapeDept(dept database.Dept, c *http.Client, db *sql.DB) {
	classSectIndex, err := fetch.Get(c, dept.Link)
	if err != nil {
		fmt.Println(err)
		return // skip if dept link is bad
	}
	if database.Insert(db, dept); err != nil {
		fmt.Println("Error inserting dept")
	}
	classes := extract.ClassIndex(classSectIndex).Extract(dept)
	for _, class := range classes {
		if err := database.Insert(db, class); err != nil {
			fmt.Println("Error inserting class")
		}
	}
	sects := extract.SectIndex(classSectIndex).Extract(classes)
	for _, sect := range sects {
		if err := database.Insert(db, sect); err != nil {
			fmt.Println("Error inserting sect")
		}
	}
}
