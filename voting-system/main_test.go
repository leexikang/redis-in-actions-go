package votingsystem

import (
	"testing"

	"github.com/gomodule/redigo/redis"
)

var conn redis.Conn

func TestMain(m *testing.M) {
	// call flag.Parse() here if TestMain uses flags
	var err error
	conn, err = redis.Dial("tcp", "localhost:6378")
	if err != nil {
		panic(err)
	}
	m.Run()
	conn.Do("FLUSHALL")
}

func TestCreatArticle(t *testing.T) {
	article := Article{
		ID:    12345,
		Title: "Example Title",
		Slug:  "Example Slug",
	}

	err := CreateArticle(conn, article)
	if err != nil {
		t.Error(err)
	}

	art, err := GetArticle(conn, article.ID)
	if err != nil {
		t.Error(err)
	}

	if art.ID != article.ID {
		t.Errorf("Expected artice with id %d got %d", article.ID, art.ID)
	}
}

func TestUpVote(t *testing.T) {
	article := Article{
		ID:    12345,
		Title: "Example Title",
		Slug:  "Example Slug",
	}

	CreateArticle(conn, article)
	err := VoteFor(conn, article.ID, 1, UpVote)
	if err != nil {
		t.Error(err)
	}

	art, err := GetArticle(conn, article.ID)
	if err != nil {
		t.Error(err)
	}

	if art.Votes != 1 {
		t.Errorf("Expected to get Vote count 1 got %d", art.Votes)
	}

	value, err := redis.Int(conn.Do("SCARD", getVotedKey(article.ID)))
	if err != nil {
		t.Error(err)
	}

	if value != 1 {
		t.Errorf("Expected to get Vote count 1 got %d", value)
	}

	value, err = redis.Int(conn.Do("ZSCORE", "score", getArticelKey(article.ID)))
	if err != nil {
		t.Error(err)
	}

	if value != 1 {
		t.Errorf("Expected to get Score 1 got %d", value)
	}
}

func TestDownVote(t *testing.T) {
	article := Article{
		ID:    12345,
		Title: "Example Title",
		Slug:  "Example Slug",
	}

	CreateArticle(conn, article)
	err := VoteFor(conn, article.ID, 1, DownVote)
	if err != nil {
		t.Error(err)
	}

	art, err := GetArticle(conn, article.ID)
	if err != nil {
		t.Error(err)
	}

	if art.Votes != -1 {
		t.Errorf("Expected to get Vote count 1 got %d", art.Votes)
	}

	value, err := redis.Int(conn.Do("SCARD", getDownVotedKey(article.ID)))
	if err != nil {
		t.Error(err)
	}

	if value != 1 {
		t.Errorf("Expected to get Vote count 1 got %d", value)
	}

	value, err = redis.Int(conn.Do("ZSCORE", "score", getArticelKey(article.ID)))
	if err != nil {
		t.Error(err)
	}

	if value != -1 {
		t.Errorf("Expected to get Score 1 got %d", value)
	}
}

func TestGetArticles(t *testing.T) {
	article := Article{
		ID:    12345,
		Title: "Example Title",
		Slug:  "Example Slug",
	}

	CreateArticle(conn, article)
	err := VoteFor(conn, article.ID, 1, UpVote)
	if err != nil {
		t.Error(err)
	}

	articles, err := GetArticles(conn, scoreTable)
	if err != nil {
		t.Error(err)
	}

	if len(articles) != 1 {
		t.Errorf("Expected to get Articles count 1 got %d", len(articles))
	}
}

func TestAddRemoveGroup(t *testing.T) {
	article := Article{
		ID:    12345,
		Title: "Example Title",
		Slug:  "Example Slug",
	}

	err := CreateArticle(conn, article)
	if err != nil {
		t.Error(err)
	}

	err = AddRemoveGroups(conn, article.ID,
		[]string{"redis"},
		[]string{},
	)

	if err != nil {
		t.Error(err)
	}

	count, err := redis.Int(conn.Do("SCARD", getGroupKey("redis")))
	if err != nil {
		t.Error(err)
	}

	if count != 1 {
		t.Errorf("Expected to get 1 article from group got %d", count)
	}

	err = AddRemoveGroups(conn, article.ID,
		[]string{},
		[]string{"redis"},
	)

	if err != nil {
		t.Error(err)
	}

	count, err = redis.Int(conn.Do("SCARD", getGroupKey("redis")))
	if err != nil {
		t.Error(err)
	}

	if count != 0 {
		t.Errorf("Expected to get 0 article from group got %d", count)
	}
}

func TestGetGropuArticles(t *testing.T) {
	article := Article{
		ID:    12345,
		Title: "Example Title",
		Slug:  "Example Slug",
	}

	err := CreateArticle(conn, article)
	if err != nil {
		t.Error(err)
	}

	err = AddRemoveGroups(conn, article.ID,
		[]string{"redis"},
		[]string{},
	)

	if err != nil {
		t.Error(err)
	}

	articles, err := GetGroupArticles(conn, "redis", "score:")
	if err != nil {
		t.Error(err)
	}

	if len(articles) != 1 {
		t.Errorf("Expected to get 1 article from group articles got %d", len(articles))
	}
}
