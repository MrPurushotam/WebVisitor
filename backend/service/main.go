package service

import (
	"database/sql"
	"log"
	"net/http"
	"sync"
	"time"

	db "github.com/MrPurushotam/web-visitor/config"
	"github.com/go-co-op/gocron/v2"
)

var (
    disableCornJob = false
    scheduler      gocron.Scheduler
    schedMu        sync.Mutex
)

func StopCornJob() {
    schedMu.Lock()
    defer schedMu.Unlock()
    disableCornJob = true
    if scheduler != nil {
        scheduler.Shutdown()
        scheduler = nil
        log.Println("CornJob scheduler stopped.")
    }
}

func EnableCornJob() {
    schedMu.Lock()
    defer schedMu.Unlock()
    if scheduler != nil {
        log.Println("CornJob scheduler already running.")
        return
    }
    disableCornJob = false
    go InitCornService()
    log.Println("CornJob scheduler started.")
}

func InitCornService() {
	schedMu.Lock()
    if scheduler != nil {
        schedMu.Unlock()
        return
    }

	s, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))
	if err != nil {
		log.Fatalf("Failed to create scheduler: %v", err)
	}

	scheduler = s
    schedMu.Unlock()
	log.Printf("Initalized Corn Job.")
	_, err = s.NewJob(
		gocron.DurationJob(6*time.Hour),
		gocron.NewTask(func() {
			if !disableCornJob {
				trackAndLogUrls("6hr")
			}
		}),
	)
	if err != nil {
		log.Fatalf("Failed to schedule 6hr job: %v", err)
	}

	_, err = s.NewJob(
		gocron.DurationJob(6*time.Hour),
		gocron.NewTask(func() {
			if !disableCornJob {
				trackAndLogUrls("12hr")
			}
		}),
	)
	if err != nil {
		log.Fatalf("Failed to schedule 12hr job: %v", err)
	}

	s.Start()
}

func trackAndLogUrls(interval string) {
	rows, err := db.DB.Query("SELECT id,url FROM urls WHERE `interval`=?", interval)
	if err != nil {
		log.Printf("Error fetching URLs for interval %s: %v", interval, err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var url string
		if err := rows.Scan(&id, &url); err != nil {
			log.Printf("Error scanning URL row with id and url as: %v, %v with error: %v", id, url, err)
			continue
		}
		status, respTime, respCode, errMsg := checkURL(url)
		_, err = db.DB.Exec(
			"INSERT INTO logs (url_id, status, response_time, response_code, error_message) VALUES (?, ?, ?, ?, ?)",
			id, status, respTime, respCode, errMsg,
		)
		if err != nil {
			log.Printf("Error inserting log for url_id %d: %v", id, err)
		}
	}
}

// Helper to check URL status
func checkURL(url string) (status string, respTime int, respCode int, errMsg sql.NullString) {
	start := time.Now()
	resp, err := http.Get(url)
	respTime = int(time.Since(start).Milliseconds())

	if err != nil {
		status = "error"
		errMsg = sql.NullString{String: err.Error(), Valid: true}
		return
	}
	defer resp.Body.Close()

	respCode = resp.StatusCode
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		status = "online"
	} else {
		status = "offline"
	}
	errMsg = sql.NullString{Valid: false}
	return
}
