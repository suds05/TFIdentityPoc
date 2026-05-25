package globaldb

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	collMemberships = "user_team_memberships"
	collRouting       = "team_storage_routing"
)

// ErrUserNotFound is returned when no membership document exists for the user ID.
var ErrUserNotFound = errors.New("user not found")

// TeamRoute maps a team to its storage tier.
type TeamRoute struct {
	TeamID        string `json:"teamId"`
	StorageTierID int    `json:"storageTierId"`
}

// DiscoverResult is the data returned by the discover API.
type DiscoverResult struct {
	TeamIDs []string    `json:"teamIds"`
	Teams   []TeamRoute `json:"teams"`
}

type membershipDoc struct {
	ID      string   `bson:"_id"`
	TeamIDs []string `bson:"teamIds"`
}

type routingDoc struct {
	ID            string `bson:"_id"`
	StorageTierID int    `bson:"storageTierId"`
}

// Store reads GlobalDB data from MongoDB.
type Store struct {
	memberships *mongo.Collection
	routing     *mongo.Collection
}

// NewStore creates a store backed by the GlobalDB MongoDB database.
func NewStore(db *mongo.Database) *Store {
	return &Store{
		memberships: db.Collection(collMemberships),
		routing:     db.Collection(collRouting),
	}
}

// Discover returns team IDs for the user and joins storage tier routing.
func (s *Store) Discover(ctx context.Context, userID string) (DiscoverResult, error) {
	var doc membershipDoc
	err := s.memberships.FindOne(ctx, bson.M{"_id": userID}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return DiscoverResult{}, ErrUserNotFound
		}
		return DiscoverResult{}, err
	}
	teamIDs := doc.TeamIDs
	if teamIDs == nil {
		teamIDs = []string{}
	}
	teams := make([]TeamRoute, 0, len(teamIDs))

	// TODO:sudhakar - we are returning the routing information along with team memberships for the user.
	// This should ideally be split to a seperate API. The Storage Tier will only require user's team memberships.
	// Some edge component doing routing may require the routing information. So seperating is better.
	for _, teamID := range teamIDs {
		var route routingDoc
		err := s.routing.FindOne(ctx, bson.M{"_id": teamID}).Decode(&route)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				continue
			}
			return DiscoverResult{}, err
		}
		teams = append(teams, TeamRoute{
			TeamID:        teamID,
			StorageTierID: route.StorageTierID,
		})
	}
	return DiscoverResult{TeamIDs: teamIDs, Teams: teams}, nil
}
