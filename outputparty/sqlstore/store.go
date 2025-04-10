package sqlstore

import (
	"log"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DB struct {
	db *gorm.DB
}

func NewDB(id string) *DB {
	db, err := SetupDatabase(id)
	if err != nil {
		log.Fatalf("Cannot set up database: %s", err)
	}
	return &DB{db: db}

}

func SetupDatabase(sid string) (*gorm.DB, error) {
	//dsn := fmt.Sprintf("smc:smcinabox@tcp(127.0.0.1:3306)/%s?charset=utf8mb4&parseTime=True&loc=Local", sid)
	//dsn := fmt.Sprintf("smc:smcinabox@tcp(mysql-container:3306)/%s?charset=utf8mb4&parseTime=True&loc=Local", sid)
	dsn := "smc:smcinabox@tcp(127.0.0.1:3306)/smc?charset=utf8mb4&parseTime=True&loc=Local"

	// Create a new GORM logger that logs only errors
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold: time.Nanosecond, // Set the threshold to a very low value
			LogLevel:      logger.Silent,   // Set log level to Silent
			Colorful:      false,           // Disable color
		},
	)

	// Open a connection to the MySQL database
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: newLogger})
	if err != nil {
		return nil, err
	}

	log.Printf("Connection to %s Database Established\n", sid)

	// Auto-migrate tables
	if err := db.AutoMigrate(&Experiment{}, &ServerShare{}); err != nil {
		return nil, err
	}

	return db, nil
}

// create server sumShare record in the server table
func (db *DB) InsertServerShare(exp_id, server_id string, shares []byte) error {
	s := ServerShare{
		Exp_ID:    exp_id,
		Server_ID: server_id,
		Shares:    shares,
	}
	result := db.db.Create(&s)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// get server's aggregated share
func (db *DB) GetSharesPerServer(exp_id, server_id string) ([]ServerShare, error) {
	var shares []ServerShare
	r := db.db.Find(&shares, "exp_id = ? and server_id = ?", exp_id, server_id)
	if r.Error != nil {
		return nil, r.Error
	}
	return shares, nil
}

// get all shares of an experiment
func (db *DB) GetSharesPerExperiment(exp_id string) ([]ServerShare, error) {
	var shares []ServerShare
	r := db.db.Find(&shares, "exp_id = ?", exp_id)
	if r.Error != nil {
		return nil, r.Error
	}

	return shares, nil
}

func (db *DB) CountSharesPerExperiment(exp_id string) int64 {
	var count int64
	db.db.Model(&ServerShare{}).Where("exp_id = ?", exp_id).Count(&count)

	return count
}

// create experiment record in the experiment tables
func (db *DB) InsertExperiment(exp_id, due1, due2 string) error {
	exp := &Experiment{
		Exp_ID:         exp_id,
		ClientShareDue: due1,
		ServerShareDue: due2,
		Completed:      false,
	}
	result := db.db.Create(&exp)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// get experiment record
func (db *DB) GetExperiment(exp_id string) (*Experiment, error) {
	var exp Experiment
	r := db.db.Find(&exp, "exp_id = ?", exp_id)
	if r.Error != nil {
		return nil, r.Error
	}
	return &exp, nil
}

// get all experiments records that server round is not completed
func (db *DB) GetAllExperiments() ([]Experiment, error) {
	var experiments []Experiment
	r := db.db.Find(&experiments, "completed=?", false)
	if r.Error != nil {
		return nil, r.Error
	}

	return experiments, nil
}

// set experiment status to completed
func (db *DB) UpdateCompletedExperiment(exp_id string) error {
	var exp Experiment
	r := db.db.Model(&exp).Where("exp_ID = ?", exp_id).Update("Completed", true)
	if r.Error != nil {
		return r.Error
	}
	return nil
}

// delete experiment record from experiment table
func (db *DB) DeleteExperiment(exp_id string) error {
	r := db.db.Delete(&Experiment{Exp_ID: exp_id})
	if r.Error != nil {
		return r.Error
	}
	return nil
}
