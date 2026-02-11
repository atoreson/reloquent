package target

import (
	"context"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/reloquent/reloquent/internal/sizing"
)

// MongoOperator implements Operator using the MongoDB driver.
type MongoOperator struct {
	client   *mongo.Client
	database string
	connStr  string
}

// NewMongoOperator creates a new MongoOperator connected to the given MongoDB instance.
func NewMongoOperator(ctx context.Context, connectionString, database string) (*MongoOperator, error) {
	opts := options.Client().ApplyURI(connectionString)
	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, fmt.Errorf("connecting to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("pinging MongoDB: %w", err)
	}

	return &MongoOperator{
		client:   client,
		database: database,
		connStr:  connectionString,
	}, nil
}

// DetectTopology determines the MongoDB deployment topology.
func (m *MongoOperator) DetectTopology(ctx context.Context) (*TopologyInfo, error) {
	info := &TopologyInfo{}

	// Run hello command
	var result bson.M
	err := m.client.Database("admin").RunCommand(ctx, bson.D{{Key: "hello", Value: 1}}).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("running hello command: %w", err)
	}

	// Detect Atlas via connection string
	info.IsAtlas = strings.Contains(m.connStr, "mongodb.net")

	// Detect topology type
	if msg, ok := result["msg"]; ok && msg == "isdbgrid" {
		info.Type = "sharded"
		// Get shard count
		var shardResult bson.M
		err := m.client.Database("config").RunCommand(ctx, bson.D{{Key: "count", Value: "shards"}}).Decode(&shardResult)
		if err == nil {
			if n, ok := shardResult["n"]; ok {
				if count, ok := n.(int32); ok {
					info.ShardCount = int(count)
				}
			}
		}
	} else if _, ok := result["setName"]; ok {
		info.Type = "replica_set"
	} else {
		info.Type = "standalone"
	}

	if info.IsAtlas {
		info.Type = "atlas"
	}

	// Get server version
	var buildInfo bson.M
	err = m.client.Database("admin").RunCommand(ctx, bson.D{{Key: "buildInfo", Value: 1}}).Decode(&buildInfo)
	if err == nil {
		if v, ok := buildInfo["version"]; ok {
			info.ServerVersion = fmt.Sprintf("%v", v)
		}
	}

	// Get storage size
	var dbStats bson.M
	err = m.client.Database(m.database).RunCommand(ctx, bson.D{{Key: "dbStats", Value: 1}}).Decode(&dbStats)
	if err == nil {
		if s, ok := dbStats["storageSize"]; ok {
			switch v := s.(type) {
			case int64:
				info.StorageBytes = v
			case int32:
				info.StorageBytes = int64(v)
			case float64:
				info.StorageBytes = int64(v)
			}
		}
	}

	return info, nil
}

// Validate checks that the target MongoDB meets the requirements of the sizing plan.
func (m *MongoOperator) Validate(ctx context.Context, plan *sizing.SizingPlan) (*ValidationResult, error) {
	result := &ValidationResult{Passed: true}

	topo, err := m.DetectTopology(ctx)
	if err != nil {
		return nil, fmt.Errorf("detecting topology: %w", err)
	}

	// Check if sharding is needed but not available
	if plan.ShardPlan != nil && plan.ShardPlan.Recommended {
		if topo.Type != "sharded" && topo.Type != "atlas" {
			result.Errors = append(result.Errors, ValidationIssue{
				Category:   "shard",
				Message:    "Sharding plan requires a sharded cluster, but target is " + topo.Type,
				Suggestion: "Deploy a sharded MongoDB cluster or use MongoDB Atlas with sharding enabled.",
			})
			result.Passed = false
		}
	}

	// Warn about standalone deployments
	if topo.Type == "standalone" {
		result.Warnings = append(result.Warnings, ValidationIssue{
			Category:   "tier",
			Message:    "Target is a standalone MongoDB instance (no replica set).",
			Suggestion: "Consider using a replica set for production migrations to ensure data durability.",
		})
	}

	return result, nil
}

// CreateCollections creates empty collections in the target database.
func (m *MongoOperator) CreateCollections(ctx context.Context, names []string) error {
	db := m.client.Database(m.database)
	for _, name := range names {
		if err := db.CreateCollection(ctx, name); err != nil {
			// Ignore "already exists" errors
			if !strings.Contains(err.Error(), "already exists") {
				return fmt.Errorf("creating collection %s: %w", name, err)
			}
		}
	}
	return nil
}

// SetupSharding configures sharding on the target database.
func (m *MongoOperator) SetupSharding(ctx context.Context, plan *sizing.ShardingPlan) error {
	if plan == nil || !plan.Recommended {
		return nil
	}

	admin := m.client.Database("admin")

	// Enable sharding on the database
	if err := admin.RunCommand(ctx, bson.D{{Key: "enableSharding", Value: m.database}}).Err(); err != nil {
		if !strings.Contains(err.Error(), "already enabled") {
			return fmt.Errorf("enabling sharding on database: %w", err)
		}
	}

	// Shard each collection
	for _, col := range plan.Collections {
		shardKey := bson.D{}
		for k, v := range col.ShardKey {
			if v == "hashed" {
				shardKey = append(shardKey, bson.E{Key: k, Value: "hashed"})
			} else {
				shardKey = append(shardKey, bson.E{Key: k, Value: 1})
			}
		}

		ns := m.database + "." + col.CollectionName
		cmd := bson.D{
			{Key: "shardCollection", Value: ns},
			{Key: "key", Value: shardKey},
		}

		if err := admin.RunCommand(ctx, cmd).Err(); err != nil {
			if !strings.Contains(err.Error(), "already sharded") {
				return fmt.Errorf("sharding collection %s: %w", col.CollectionName, err)
			}
		}
	}

	return nil
}

// DisableBalancer stops the MongoDB balancer during migration.
func (m *MongoOperator) DisableBalancer(ctx context.Context) error {
	return m.client.Database("admin").RunCommand(ctx, bson.D{
		{Key: "balancerStop", Value: 1},
	}).Err()
}

// EnableBalancer starts the MongoDB balancer after migration.
func (m *MongoOperator) EnableBalancer(ctx context.Context) error {
	return m.client.Database("admin").RunCommand(ctx, bson.D{
		{Key: "balancerStart", Value: 1},
	}).Err()
}

// DropCollections drops the specified collections from the target database.
func (m *MongoOperator) DropCollections(ctx context.Context, names []string) error {
	db := m.client.Database(m.database)
	for _, name := range names {
		if err := db.Collection(name).Drop(ctx); err != nil {
			return fmt.Errorf("dropping collection %s: %w", name, err)
		}
	}
	return nil
}

// CountDocuments returns the number of documents in a collection.
func (m *MongoOperator) CountDocuments(ctx context.Context, collection string) (int64, error) {
	count, err := m.client.Database(m.database).Collection(collection).CountDocuments(ctx, bson.D{})
	if err != nil {
		return 0, fmt.Errorf("counting documents in %s: %w", collection, err)
	}
	return count, nil
}

// SampleDocuments returns n random documents from a collection using $sample.
func (m *MongoOperator) SampleDocuments(ctx context.Context, collection string, n int) ([]map[string]interface{}, error) {
	pipeline := bson.A{bson.D{{Key: "$sample", Value: bson.D{{Key: "size", Value: n}}}}}
	cursor, err := m.client.Database(m.database).Collection(collection).Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("sampling documents from %s: %w", collection, err)
	}
	defer cursor.Close(ctx)

	var results []map[string]interface{}
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("decoding sample document: %w", err)
		}
		row := make(map[string]interface{}, len(doc))
		for k, v := range doc {
			row[k] = v
		}
		results = append(results, row)
	}
	return results, cursor.Err()
}

// AggregateSum returns the SUM of a numeric field across all documents.
func (m *MongoOperator) AggregateSum(ctx context.Context, collection, field string) (float64, error) {
	pipeline := bson.A{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "total", Value: bson.D{{Key: "$sum", Value: "$" + field}}},
		}}},
	}
	cursor, err := m.client.Database(m.database).Collection(collection).Aggregate(ctx, pipeline)
	if err != nil {
		return 0, fmt.Errorf("aggregating sum on %s.%s: %w", collection, field, err)
	}
	defer cursor.Close(ctx)

	if cursor.Next(ctx) {
		var result bson.M
		if err := cursor.Decode(&result); err != nil {
			return 0, fmt.Errorf("decoding sum result: %w", err)
		}
		switch v := result["total"].(type) {
		case float64:
			return v, nil
		case int32:
			return float64(v), nil
		case int64:
			return float64(v), nil
		}
	}
	return 0, nil
}

// AggregateCountDistinct returns the number of distinct values for a field.
func (m *MongoOperator) AggregateCountDistinct(ctx context.Context, collection, field string) (int64, error) {
	pipeline := bson.A{
		bson.D{{Key: "$group", Value: bson.D{{Key: "_id", Value: "$" + field}}}},
		bson.D{{Key: "$count", Value: "count"}},
	}
	cursor, err := m.client.Database(m.database).Collection(collection).Aggregate(ctx, pipeline)
	if err != nil {
		return 0, fmt.Errorf("counting distinct %s.%s: %w", collection, field, err)
	}
	defer cursor.Close(ctx)

	if cursor.Next(ctx) {
		var result bson.M
		if err := cursor.Decode(&result); err != nil {
			return 0, fmt.Errorf("decoding count distinct result: %w", err)
		}
		switch v := result["count"].(type) {
		case int32:
			return int64(v), nil
		case int64:
			return v, nil
		}
	}
	return 0, nil
}

// CreateIndex creates a single index on a collection.
func (m *MongoOperator) CreateIndex(ctx context.Context, collection string, index IndexDefinition) error {
	keys := bson.D{}
	for _, k := range index.Keys {
		keys = append(keys, bson.E{Key: k.Field, Value: k.Order})
	}

	opts := options.Index()
	if index.Name != "" {
		opts.SetName(index.Name)
	}
	if index.Unique {
		opts.SetUnique(true)
	}

	model := mongo.IndexModel{
		Keys:    keys,
		Options: opts,
	}

	_, err := m.client.Database(m.database).Collection(collection).Indexes().CreateOne(ctx, model)
	if err != nil {
		return fmt.Errorf("creating index on %s: %w", collection, err)
	}
	return nil
}

// CreateIndexes creates multiple indexes across collections.
func (m *MongoOperator) CreateIndexes(ctx context.Context, indexes []CollectionIndex) error {
	for _, ci := range indexes {
		if err := m.CreateIndex(ctx, ci.Collection, ci.Index); err != nil {
			return err
		}
	}
	return nil
}

// ListIndexBuildProgress queries currentOp for active index build operations.
func (m *MongoOperator) ListIndexBuildProgress(ctx context.Context) ([]IndexBuildStatus, error) {
	cmd := bson.D{
		{Key: "currentOp", Value: true},
		{Key: "active", Value: true},
	}
	var result bson.M
	err := m.client.Database("admin").RunCommand(ctx, cmd).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("querying currentOp: %w", err)
	}

	var statuses []IndexBuildStatus
	inprog, ok := result["inprog"]
	if !ok {
		return statuses, nil
	}

	ops, ok := inprog.(bson.A)
	if !ok {
		return statuses, nil
	}

	for _, op := range ops {
		doc, ok := op.(bson.M)
		if !ok {
			continue
		}
		desc, _ := doc["desc"].(string)
		if !strings.Contains(desc, "Index") {
			// Also check the command field
			cmdDoc, _ := doc["command"].(bson.M)
			if cmdDoc == nil {
				continue
			}
			if _, hasCreateIndexes := cmdDoc["createIndexes"]; !hasCreateIndexes {
				continue
			}
		}

		ns, _ := doc["ns"].(string)
		msg, _ := doc["msg"].(string)
		var progress float64
		if p, ok := doc["progress"].(bson.M); ok {
			done, _ := p["done"].(int64)
			total, _ := p["total"].(int64)
			if total > 0 {
				progress = float64(done) / float64(total) * 100
			}
		}

		statuses = append(statuses, IndexBuildStatus{
			Collection: ns,
			IndexName:  desc,
			Phase:      "building",
			Progress:   progress,
			Message:    msg,
		})
	}

	return statuses, nil
}

// SetWriteConcern sets the default write concern on the database.
func (m *MongoOperator) SetWriteConcern(ctx context.Context, w string, journal bool) error {
	wc := bson.D{{Key: "w", Value: w}, {Key: "j", Value: journal}}
	cmd := bson.D{
		{Key: "setDefaultRWConcern", Value: 1},
		{Key: "defaultWriteConcern", Value: wc},
	}
	if err := m.client.Database("admin").RunCommand(ctx, cmd).Err(); err != nil {
		return fmt.Errorf("setting write concern: %w", err)
	}
	return nil
}

// Close disconnects from MongoDB.
func (m *MongoOperator) Close(ctx context.Context) error {
	return m.client.Disconnect(ctx)
}
