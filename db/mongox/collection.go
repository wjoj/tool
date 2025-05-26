package mongox

import (
	"context"
	"reflect"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Collection struct {
	col             *mongo.Collection
	dbName, colName string
}

func (c *Collection) SwitchCollection(name string) *Collection {
	return &Collection{
		colName: name,
		col:     c.col.Database().Collection(name),
	}
}

func (c *Collection) Collection() *mongo.Collection {
	return c.col
}

func (c *Collection) Insert(d ...any) ([]any, error) {
	if len(d) == 1 {
		res, err := c.col.InsertOne(context.Background(), d[0])
		if err != nil {
			return nil, err
		}
		return []any{res.InsertedID}, nil
	} else {
		res, err := c.col.InsertMany(context.Background(), d)
		if err != nil {
			return nil, err
		}
		return res.InsertedIDs, nil
	}
}

func (c *Collection) Update(filter bson.M, d ...any) (int64, error) {
	if len(d) == 1 {
		res, err := c.col.UpdateOne(context.Background(), filter, d[0])
		if err != nil {
			return 0, err
		}
		if res.ModifiedCount == res.UpsertedCount {
			return res.UpsertedCount, nil
		}
		return res.ModifiedCount - res.UpsertedCount, nil
	} else {
		res, err := c.col.UpdateMany(context.Background(), filter, d)
		if err != nil {
			return 0, err
		}
		if res.ModifiedCount == res.UpsertedCount {
			return res.UpsertedCount, nil
		}
		return res.ModifiedCount - res.UpsertedCount, nil
	}
}

func (c *Collection) Delete(filter bson.M, isMany bool) error {
	if isMany {
		_, err := c.col.DeleteMany(context.Background(), filter)
		if err != nil {
			return err
		}
	}
	_, err := c.col.DeleteOne(context.Background(), filter)
	return err
}

func (c *Collection) Find(sel string, filter bson.M, offset, limit int64, out any) error {
	sels := strings.Split(sel, ",")
	sls := make(bson.D, len(sels))
	for _, s := range sels {
		sls = append(sls, bson.E{Key: s, Value: 1})
	}
	switch reflect.TypeOf(out).Kind() {
	case reflect.Array, reflect.Slice:
		opts := []options.Lister[options.FindOptions]{}
		if len(sls) != 0 {
			opts = append(opts, options.Find().SetProjection(sls))
		}
		if limit != 0 {
			opts = append(opts, options.Find().SetLimit(limit))
		}
		if offset != 0 {
			opts = append(opts, options.Find().SetSkip(offset))
		}
		cur, err := c.col.Find(context.Background(), filter, opts...)
		if err != nil {
			return err
		}
		err = cur.All(context.Background(), out)
		if err != nil {
			return err
		}
		return cur.Close(context.Background())
	}
	opts := []options.Lister[options.FindOneOptions]{}
	if len(sls) != 0 {
		opts = append(opts, options.FindOne().SetProjection(sls))
	}
	return c.col.FindOne(context.Background(), filter, opts...).Decode(out)
}

func (c *Collection) Aggregate(pipeline bson.D, v any) error {
	cur, err := c.col.Aggregate(context.Background(), pipeline)
	if err != nil {
		return err
	}
	err = cur.All(context.Background(), v)
	if err != nil {
		return err
	}
	return cur.Close(context.Background())
}

func (c *Collection) Count(filter bson.D) (int64, error) {
	return c.col.CountDocuments(context.Background(), filter)
}
