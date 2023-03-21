package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

// definisi struktur model data
type Order struct {
	ID        uint      `gorm:"primaryKey:autoIncrement"`
	RequestID uint      `gorm:"not null"`
	Customer  string    `gorm:"type:varchar(100);not null"`
	Quantity  uint      `gorm:"not null"`
	Price     float64   `gorm:"not null"`
	CreatedAt time.Time `json:"created_at" gorm:"type:timestamp;not null"`
}

func (Order) TableName() string {
	return "order"
}

func main() {
	// buka koneksi ke database MySQL
	db, err := gorm.Open("mysql", "dev:1@tcp(127.0.0.1:3306)/person?charset=utf8mb4&parseTime=True&loc=Local")
	if err != nil {
		panic("Failed to connect to database!")
	}
	db.DB().SetMaxIdleConns(10)
	db.DB().SetMaxOpenConns(100)
	defer db.Close()

	// konfigurasi GIN framework
	router := gin.Default()

	// handler untuk route POST /orders
	router.POST("/orders", func(c *gin.Context) {
		// baca data dari request body JSON
		var payload struct {
			RequestID uint    `json:"request_id"`
			Data      []Order `json:"data"`
		}
		if err := c.BindJSON(&payload); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
			return
		}

		// validasi jumlah data
		if len(payload.Data) > 1000 {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "too many data"})
			return
		}

		// simpan data ke database dengan GORM
		start := time.Now()
		var newORder []Order
		for _, order := range payload.Data {
			order.RequestID = payload.RequestID
			newORder = append(newORder, order)
		}

		if err := createUsersConcurrent(newORder, db); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		elapsed := time.Since(start)

		// kirim data sebagai response JSON
		c.JSON(http.StatusCreated, gin.H{
			"success": true,
			"time":    elapsed.Milliseconds(),
		})
	})

	// generate data dummy
	// generateData()

	router.Run(":8080")
}

// generate dummy data
func generateData() []Order {
	rand.Seed(time.Now().UnixNano())

	order := make([]Order, 1000)

	for i := 0; i < 1000; i++ {
		n := 10000
		randomInt := rand.Intn(n)
		// Set up the faker library
		gofakeit.Seed(0)

		order[i] = Order{
			Customer:  gofakeit.Name(), // Generate a random person's name
			Quantity:  uint(randomInt),
			Price:     10,
			CreatedAt: time.Now(),
		}
	}

	file, err := os.Create("dummy1000.json")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Write the data
	encoder := json.NewEncoder(file)
	err = encoder.Encode(order)
	if err != nil {
		panic(err)
	}

	return order
}

func createUsersConcurrent(orders []Order, db *gorm.DB) error {
	var wg sync.WaitGroup
	var err error
	for i := 0; i < len(orders); i++ {
		wg.Add(1)
		go func(order Order) {
			defer wg.Done()
			if err := db.Create(&order).Error; err != nil {
				err = fmt.Errorf("%v\n%v", err, order)
			}
		}(orders[i])
	}
	wg.Wait()
	return err
}
