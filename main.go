package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/IBM/sarama"
	_ "github.com/lib/pq"
)

type CVERecord struct {
	Containers struct {
		CNA struct {
			Affected []struct {
				Product  string `json:"product"`
				Vendor   string `json:"vendor"`
				Versions []struct {
					Status  string `json:"status"`
					Version string `json:"version"`
				} `json:"versions"`
			} `json:"affected"`
			Descriptions []struct {
				Lang  string `json:"lang"`
				Value string `json:"value"`
			} `json:"descriptions"`
			ProblemTypes []struct {
				Descriptions []struct {
					Description string `json:"description"`
					Lang        string `json:"lang"`
					Type        string `json:"type"`
				} `json:"descriptions"`
			} `json:"problemTypes"`
			ProviderMetadata struct {
				DateUpdated string `json:"dateUpdated"`
				OrgID       string `json:"orgId"`
				ShortName   string `json:"shortName"`
			} `json:"providerMetadata"`
			References []struct {
				Name string   `json:"name"`
				Tags []string `json:"tags"`
				URL  string   `json:"url"`
			} `json:"references"`
			XLegacyV4Record struct {
				CVEDataMeta struct {
					Assigner string `json:"ASSIGNER"`
					ID       string `json:"ID"`
					State    string `json:"STATE"`
				} `json:"CVE_data_meta"`
				Affects struct {
					Vendor struct {
						VendorData []struct {
							Product struct {
								ProductData []struct {
									ProductName string `json:"product_name"`
									Version     struct {
										VersionData []struct {
											VersionValue string `json:"version_value"`
										} `json:"version_data"`
									} `json:"version"`
								} `json:"product_data"`
							} `json:"product"`
							VendorName string `json:"vendor_name"`
						} `json:"vendor_data"`
					} `json:"vendor"`
				} `json:"affects"`
				DataFormat  string `json:"data_format"`
				DataType    string `json:"data_type"`
				DataVersion string `json:"data_version"`
				Description struct {
					DescriptionData []struct {
						Lang  string `json:"lang"`
						Value string `json:"value"`
					} `json:"description_data"`
				} `json:"description"`
				ProblemType struct {
					ProblemTypeData []struct {
						Description []struct {
							Lang  string `json:"lang"`
							Value string `json:"value"`
						} `json:"description"`
					} `json:"problemtype_data"`
				} `json:"problemtype"`
				References struct {
					ReferenceData []struct {
						Name      string `json:"name"`
						Refsource string `json:"refsource"`
						URL       string `json:"url"`
					} `json:"reference_data"`
				} `json:"references"`
			} `json:"x_legacyV4Record"`
		} `json:"cna"`
	} `json:"containers"`
	CveMetadata struct {
		AssignerOrgID     string `json:"assignerOrgId"`
		AssignerShortName string `json:"assignerShortName"`
		CveID             string `json:"cveId"`
		DatePublished     string `json:"datePublished"`
		DateReserved      string `json:"dateReserved"`
		DateUpdated       string `json:"dateUpdated"`
		State             string `json:"state"`
		Version           string `json:"version"`
	} `json:"cveMetadata"`
	DataType    string `json:"dataType"`
	DataVersion string `json:"dataVersion"`
}

const (
	batchSize = 100
	topic     = "cve"
)

func main() {

	// Database configuration
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatalf("Error opening database: %v\n", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	err = db.Ping()
	if err != nil {
		log.Fatalf("Error pinging database: %v", err)
	}
	log.Println("Successfully connected to the database")

	// Create the "cve" schema if it doesn't exist
	_, err = db.Exec(`CREATE SCHEMA IF NOT EXISTS cve`)
	if err != nil {
		log.Fatalf("Error creating schema: %v", err)
	}

	// Create the cve table if it doesn't exist in the "cve" schema
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS cve.cve (
			id TEXT,
			version INTEGER,
			assigner_org_id TEXT,
			assigner_short_name TEXT,
			date_published TEXT,
			date_reserved TEXT,
			date_updated TEXT,
			state TEXT,
			data JSONB,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (id, version)
		)
	`)
	if err != nil {
		fmt.Println("Error creating cve table:", err)
		os.Exit(1)
	} else {
		fmt.Println("cve table created successfully")
	}

	// Create indexes
	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_cve_data_gin ON cve.cve USING GIN (data);
		CREATE INDEX IF NOT EXISTS idx_cve_metadata_gin ON cve.cve USING GIN ((data->'cveMetadata'));
	`)
	if err != nil {
		log.Fatalf("Error creating indexes: %v", err)
	}

	fmt.Println("Indexes created successfully")

	// Kafka configuration
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	config.Version = sarama.V2_8_0_0

	brokers := strings.Split(os.Getenv("KAFKA_BROKERS"), ",")
	if len(brokers) == 0 {
		log.Fatal("KAFKA_BROKERS environment variable is not set")
	}

	consumer, err := sarama.NewConsumerGroup(brokers, "cve-consumer-group", config)
	if err != nil {
		log.Fatalf("Error creating consumer group client: %v", err)
	}
	defer consumer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	wg.Add(1)

	handler := &ConsumerGroupHandler{
		db:        db,
		batchSize: batchSize,
	}

	go func() {
		defer wg.Done()
		for {
			if err := consumer.Consume(ctx, []string{topic}, handler); err != nil {
				log.Printf("Error from consumer: %v", err)
			}
			if ctx.Err() != nil {
				return
			}
		}
	}()

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, os.Interrupt)
	select {
	case <-sigterm:
		log.Println("Received termination signal, shutting down...")
	case <-ctx.Done():
	}
	cancel()
	wg.Wait()
}

type ConsumerGroupHandler struct {
	db        *sql.DB
	batchSize int
	mu        sync.Mutex
	batch     []CVERecord
}

func (h *ConsumerGroupHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (h *ConsumerGroupHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (h *ConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		var cveRecord CVERecord
		if err := json.Unmarshal(message.Value, &cveRecord); err != nil {
			log.Printf("Error parsing CVE JSON data: %v", err)
			continue
		}

		h.mu.Lock()
		h.batch = append(h.batch, cveRecord)
		if len(h.batch) >= h.batchSize {
			batch := h.batch
			h.batch = nil
			h.mu.Unlock()
			if err := h.processBatch(batch); err != nil {
				log.Printf("Error processing batch: %v", err)
			}
		} else {
			h.mu.Unlock()
		}

		session.MarkMessage(message, "")
	}
	return nil
}

func (h *ConsumerGroupHandler) processBatch(batch []CVERecord) error {
	tx, err := h.db.Begin()
	if err != nil {
		return fmt.Errorf("error starting transaction: %v", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO cve.cve (id, version, assigner_org_id, assigner_short_name, date_published, date_reserved, date_updated, state, data)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id, version) DO UPDATE SET
			assigner_org_id = EXCLUDED.assigner_org_id,
			assigner_short_name = EXCLUDED.assigner_short_name,
			date_published = EXCLUDED.date_published,
			date_reserved = EXCLUDED.date_reserved,
			date_updated = EXCLUDED.date_updated,
			state = EXCLUDED.state,
			data = EXCLUDED.data
	`)
	if err != nil {
		return fmt.Errorf("error preparing statement: %v", err)
	}
	defer stmt.Close()

	for _, record := range batch {
		data, err := json.Marshal(record)
		if err != nil {
			log.Printf("Error marshaling CVE record: %v", err)
			continue
		}

		_, err = stmt.Exec(
			record.CveMetadata.CveID,
			getVersion(record.CveMetadata.DateUpdated),
			record.CveMetadata.AssignerOrgID,
			record.CveMetadata.AssignerShortName,
			record.CveMetadata.DatePublished,
			record.CveMetadata.DateReserved,
			record.CveMetadata.DateUpdated,
			record.CveMetadata.State,
			data,
		)
		if err != nil {
			log.Printf("Error inserting/updating CVE record %s: %v", record.CveMetadata.CveID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing transaction: %v", err)
	}

	log.Printf("Processed batch of %d records", len(batch))
	return nil
}

func getVersion(dateUpdated string) int {
	// Implement a version calculation based on the dateUpdated
	// This is a simple example; you might want to implement a more sophisticated versioning system
	t, err := time.Parse(time.RFC3339, dateUpdated)
	if err != nil {
		return 1
	}
	return int(t.Unix())
}
