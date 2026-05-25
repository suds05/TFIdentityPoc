package storagedb

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const collTeams = "teams"

// ErrTeamNotFound is returned when the team is not provisioned on this storage tier.
var ErrTeamNotFound = errors.New("team not found")

// Folder is a folder within a team.
type Folder struct {
	FolderID string `json:"folderId" bson:"folderId"`
	Name     string `json:"name" bson:"name"`
}

// ListFoldersResult is the list folders API response body.
type ListFoldersResult struct {
	TeamID  string   `json:"teamId"`
	Folders []Folder `json:"folders"`
}

type teamDoc struct {
	ID      string   `bson:"_id"`
	Folders []Folder `bson:"folders"`
}

// Store reads team data from a storage tier MongoDB database.
type Store struct {
	teams *mongo.Collection
}

// NewStore creates a store for the given storage tier database.
func NewStore(db *mongo.Database) *Store {
	return &Store{teams: db.Collection(collTeams)}
}

// ListFolders returns folders for a team provisioned on this tier.
func (s *Store) ListFolders(ctx context.Context, teamID string) (ListFoldersResult, error) {
	var doc teamDoc
	err := s.teams.FindOne(ctx, bson.M{"_id": teamID}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return ListFoldersResult{}, ErrTeamNotFound
		}
		return ListFoldersResult{}, err
	}
	folders := doc.Folders
	if folders == nil {
		folders = []Folder{}
	}
	return ListFoldersResult{TeamID: teamID, Folders: folders}, nil
}
