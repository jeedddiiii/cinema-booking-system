package config

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Config struct {
	Port        string
	MongoURI    string
	RedisHost   string
	RedisPort   string
	KafkaBroker string
	KafkaTopic  string
}

var (
	MongoDB     *mongo.Database
	RedisClient *redis.Client
	KafkaWriter *kafka.Writer
	AppConfig   *Config
)

func LoadConfig() *Config {
	godotenv.Load()

	config := &Config{
		Port:        getEnv("PORT", "8080"),
		MongoURI:    getEnv("MONGO_URI", "mongodb://admin:password@localhost:27017/cinema_db?authSource=admin"),
		RedisHost:   getEnv("REDIS_HOST", "localhost"),
		RedisPort:   getEnv("REDIS_PORT", "6379"),
		KafkaBroker: getEnv("KAFKA_BROKER", "localhost:9092"),
		KafkaTopic:  getEnv("KAFKA_TOPIC", "audit-logs"),
	}

	AppConfig = config
	return config
}

func InitMongoDB(uri string) (*mongo.Database, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	log.Println("✅ Connected to MongoDB")
	MongoDB = client.Database("cinema_db")
	return MongoDB, nil
}

func InitRedis(host, port string) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     host + ":" + port,
		Password: "",
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := client.Ping(ctx).Result(); err != nil {
		log.Printf("⚠️ Redis connection failed: %v", err)
	} else {
		log.Println("✅ Connected to Redis")
	}

	RedisClient = client
	return client
}

func InitKafka(broker, topic string) *kafka.Writer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(broker),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
	}

	log.Printf("✅ Kafka producer initialized for topic: %s", topic)
	KafkaWriter = writer
	return writer
}

func CloseConnections() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if MongoDB != nil {
		if err := MongoDB.Client().Disconnect(ctx); err != nil {
			log.Printf("Error disconnecting MongoDB: %v", err)
		}
	}

	if RedisClient != nil {
		if err := RedisClient.Close(); err != nil {
			log.Printf("Error closing Redis: %v", err)
		}
	}

	if KafkaWriter != nil {
		if err := KafkaWriter.Close(); err != nil {
			log.Printf("Error closing Kafka: %v", err)
		}
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
