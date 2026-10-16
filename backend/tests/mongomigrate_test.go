package tests

import (
	"testing"

	"github.com/divyanshmehta355/aurahub/backend/db"
	"github.com/divyanshmehta355/aurahub/backend/internal/mongomigrate"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestMappedUUIDIsStableAndCollectionScoped(t *testing.T) {
	id := primitive.NewObjectID()
	first := mongomigrate.MappedUUID("users", id)
	if first != mongomigrate.MappedUUID("users", id) {
		t.Fatal("mapping changed for the same collection and ObjectId")
	}
	if first == mongomigrate.MappedUUID("videos", id) {
		t.Fatal("different collections must use different UUID mappings")
	}
	if first == uuid.Nil {
		t.Fatal("mapped UUID must not be nil")
	}
}

func TestSortCommentsInParentOrder(t *testing.T) {
	parent, child := primitive.NewObjectID(), primitive.NewObjectID()
	rows := []db.Comment{
		{ID: mongomigrate.PgUUID("comments", child)},
		{ID: mongomigrate.PgUUID("comments", parent)},
	}
	sorted, skipped, err := mongomigrate.SortComments(rows, map[primitive.ObjectID]primitive.ObjectID{child: parent}, []primitive.ObjectID{child, parent}, nil)
	if err != nil {
		t.Fatalf("sortComments returned an error: %v", err)
	}
	if len(skipped) != 0 {
		t.Fatalf("unexpected skipped comments: %v", skipped)
	}
	if sorted[0].ID != mongomigrate.PgUUID("comments", parent) || sorted[1].ID != mongomigrate.PgUUID("comments", child) {
		t.Fatal("parent comment must be inserted before its reply")
	}
	if sorted[1].ParentCommentID != mongomigrate.PgUUID("comments", parent) {
		t.Fatal("reply parent reference was not preserved")
	}
}

func TestSortCommentsSkipsMissingParentAndRejectsCycle(t *testing.T) {
	first, second, missing := primitive.NewObjectID(), primitive.NewObjectID(), primitive.NewObjectID()
	rows := []db.Comment{{ID: mongomigrate.PgUUID("comments", first)}, {ID: mongomigrate.PgUUID("comments", second)}}
	sorted, skipped, err := mongomigrate.SortComments(rows, map[primitive.ObjectID]primitive.ObjectID{first: missing, second: first}, []primitive.ObjectID{first, second}, nil)
	if err != nil {
		t.Fatalf("missing parent should be reported as a skipped orphan: %v", err)
	}
	if len(sorted) != 0 || len(skipped) != 2 {
		t.Fatalf("expected the orphan and its reply to be skipped; got %d imported and %d skipped", len(sorted), len(skipped))
	}
	if _, _, err := mongomigrate.SortComments(rows, map[primitive.ObjectID]primitive.ObjectID{first: second, second: first}, []primitive.ObjectID{first, second}, nil); err == nil {
		t.Fatal("cyclic parents should fail validation")
	}
}

func TestIDArrayAcceptsObjectIdsAndRejectsMalformedValues(t *testing.T) {
	first, second := primitive.NewObjectID(), primitive.NewObjectID()
	got, err := mongomigrate.IDArray(bson.M{"ids": bson.A{first, second.Hex()}}, "ids")
	if err != nil {
		t.Fatalf("idArray returned an error: %v", err)
	}
	if len(got) != 2 || got[0] != first || got[1] != second {
		t.Fatal("idArray did not preserve ids and order")
	}
	if _, err := mongomigrate.IDArray(bson.M{"ids": bson.A{"invalid"}}, "ids"); err == nil {
		t.Fatal("malformed ObjectId should fail validation")
	}
}
