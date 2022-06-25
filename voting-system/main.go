package votingsystem

import (
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
)

type Article struct {
	ID    int    `redis:"id"`
	Title string `redis:"title"`
	Slug  string `redis:"slug"`
	Votes int    `redis:"votes"`
}

const (
	scoreTable = "score"
)

type Vote string

const (
	UpVote   Vote = "UPVOTE"
	DownVote Vote = "DOWNVOTE"
)

func CreateArticle(conn redis.Conn, article Article) error {
	_, err := conn.Do("HSET", redis.Args{}.Add(getArticelKey(int(article.ID))).AddFlat(article)...)
	if err != nil {
		return err
	}

	_, err = conn.Do("ZADD", "score", 0, getArticelKey(article.ID))
	if err != nil {
		return err
	}

	_, err = conn.Do("ZADD", "time", time.Now().Unix(), getArticelKey(article.ID))
	if err != nil {
		return err
	}

	return nil
}

func GetArticle(conn redis.Conn, id int) (*Article, error) {
	values, err := redis.Values(conn.Do("HGETALL", getArticelKey(id)))
	if err != nil {
		return nil, err
	}
	var article Article
	redis.ScanStruct(values, &article)
	return &article, nil
}

func VoteFor(conn redis.Conn, articleId, userId int, vote Vote) error {
	oneWeekBefore := time.Now().Add(-(time.Hour * 24 * 7))
	createdAt, err := redis.Int64(conn.Do("ZSCORE", "time", getArticelKey(articleId)))
	if err != nil {
		return err
	}

	if createdAt < oneWeekBefore.Unix() {
		return errors.New("can't vote for article older than a week")
	}

	if vote == UpVote {
		if err = upVote(conn, articleId, userId); err != nil {
			return err
		}

		return nil
	}

	if err = downVote(conn, articleId, userId); err != nil {
		return err
	}

	return nil
}

func upVote(conn redis.Conn, articleId, userId int) error {
	exist, err := redis.Bool(conn.Do("SMOVE", getDownVotedKey(articleId), getVotedKey(articleId), getUserKey(userId)))
	if err != nil {
		return err
	}

	if !exist {
		_, err = conn.Do("SADD", getVotedKey(articleId), getUserKey(userId))
		if err != nil {
			return err
		}
	}

	_, err = conn.Do("HINCRBY", getArticelKey(articleId), "votes", 1)
	if err != nil {
		return err
	}

	_, err = conn.Do("ZINCRBY", scoreTable, "1", getArticelKey(articleId))
	if err != nil {
		return err
	}

	return nil
}

func downVote(conn redis.Conn, articleId, userId int) error {
	exist, err := redis.Bool(conn.Do("SMOVE", getVotedKey(articleId), getDownVotedKey(articleId), getUserKey(userId)))
	if err != nil {
		return err
	}

	if !exist {
		_, err = conn.Do("SADD", getDownVotedKey(articleId), getUserKey(userId))
		if err != nil {
			return err
		}
	}

	_, err = conn.Do("HINCRBY", getArticelKey(articleId), "votes", -1)
	if err != nil {
		return err
	}

	_, err = conn.Do("ZINCRBY", scoreTable, "-1", getArticelKey(articleId))
	if err != nil {
		return err
	}

	return nil
}

func GetArticles(conn redis.Conn, order string) ([]Article, error) {
	var articles []Article
	var articleKeys []string
	var err error
	if order == "" {
		articleKeys, err = redis.Strings(conn.Do("ZREVRANGE", scoreTable, 0, -1))
		if err != nil {
			return nil, err
		}
	} else {
		articleKeys, err = redis.Strings(conn.Do("ZREVRANGE", order, 0, -1))
		if err != nil {
			return nil, err
		}
	}

	for _, key := range articleKeys {
		id, err := idFromArticleKey(key)
		if err != nil {
			return articles, err
		}

		var article *Article
		if article, err = GetArticle(conn, id); err != nil {
			return articles, err
		}

		articles = append(articles, *article)
	}

	return articles, err
}

func GetGroupArticles(conn redis.Conn, gropuID string, order string) ([]Article, error) {
	var articles []Article
	if order == "" {
		order = "socre:"
	}

	key := order + gropuID
	exist, err := redis.Bool(conn.Do("EXISTS", key))
	if err != nil {
		log.Print(err)
		return articles, err
	}

	if !exist {
		_, err := conn.Do("ZINTERSTORE", key, 2, getGroupKey(gropuID), scoreTable, "AGGREGATE", "MAX")
		if err != nil {
			return articles, err
		}

		_, err = conn.Do("EXPIRE", key, "60")
		if err != nil {
			return articles, err
		}
	}

	return GetArticles(conn, key)
}

func AddRemoveGroups(conn redis.Conn, articleId int, toAdd, toRemove []string) error {
	articleKey := getArticelKey(articleId)
	for _, id := range toAdd {
		_, err := conn.Do("SADD", getGroupKey(id), articleKey)
		if err != nil {
			return err
		}
	}

	for _, key := range toRemove {
		_, err := conn.Do("SREM", getGroupKey(key), articleKey)
		if err != nil {
			return err
		}
	}

	return nil
}

func getArticelKey(id int) string {
	return "article:" + strconv.Itoa(id)
}

func getUserKey(id int) string {
	return "user:" + strconv.Itoa(id)
}

func getVotedKey(id int) string {
	return "voated:" + strconv.Itoa(id)
}

func getDownVotedKey(id int) string {
	return "down-vote:" + strconv.Itoa(id)
}
func getGroupKey(id string) string {
	return "group:" + id
}

func idFromArticleKey(key string) (int, error) {
	splitted := strings.Split(key, ":")
	if len(splitted) < 2 {
		return 0, errors.New("invalid Key format")
	}

	id, err := strconv.Atoi(splitted[1])
	if err != nil {
		return id, err
	}

	return id, nil
}
