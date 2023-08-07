package redis

import (
	"fmt"
	"go.uber.org/zap"
	"math"
	"time"

	"github.com/go-redis/redis"
)

const (
	OneWeekInSeconds         = 7 * 24 * 3600
	VoteScore        float64 = 432
	PostPerAge               = 20
)

/*
投票算法：http://www.ruanyifeng.com/blog/2012/03/ranking_algorithm_reddit.html



*/

/*
	PostVote 为帖子投票

投票分为四种情况：1.投赞成票 2.投反对票 3.取消投票 4.反转投票

记录文章参与投票的人
更新文章分数：赞成票要加分；反对票减分

v=1时，有两种情况

	1.之前没投过票，现在要投赞成票
	2.之前投过反对票，现在要改为赞成票

v=0时，有两种情况

	1.之前投过赞成票，现在要取消
	2.之前投过反对票，现在要取消

v=-1时，有两种情况

	1.之前没投过票，现在要投反对票
	2.之前投过赞成票，现在要改为反对票
*/
func PostVote(postID, userID string, v float64) (err error) {
	//if float64(time.Now().Unix())-postTime > OneWeekInSeconds {
	//	zap.L().Error("距离帖子发布已经一周了，不能在进行投票了")
	//	return ErrorVoteTimeExpire
	//}
	// 判断是否已经投过票
	flag, err := Client.Get("bluebell:post:IsPost:" + postID + userID).Result()
	if flag == "已投票" {
		zap.L().Error("该用户已经投票了")
		return err
	}
	key := KeyPostVotedZSetPrefix + postID
	ov := Client.ZScore(key, userID).Val() // 获取当前分数
	fmt.Println(ov)
	diffAbs := v
	fmt.Println(diffAbs)
	pipeline := Client.TxPipeline()
	pipeline.ZAdd(key, redis.Z{ // 记录已投票
		Score:  v,
		Member: userID,
	})
	pipeline.ZIncrBy(KeyPostScoreZSet, VoteScore*v, postID) // 更新分数

	switch v {
	case 1:
		// 取消投票 ov=1/-1 v=0
		// 投票数-1
		pipeline.HIncrBy(KeyPostInfoHashPrefix+postID, "votes", 1)
	case -1:
		// 新增投票 ov=0 v=1/-1
		// 投票数+1
		pipeline.HIncrBy(KeyPostInfoHashPrefix+postID, "votes", -1)
	default:
		// 已经投过票了
		zap.L().Error("报错了，传入的v不正确")
		return ErrorVoted
	}
	_ = pipeline.Set("bluebell:post:IsPost:"+postID+userID, "已投票", 7*24*time.Hour)

	_, err = pipeline.Exec()

	return
}

// CreatePost 使用hash存储帖子信息
func CreatePost(postID, userID, title, summary, communityName string) (err error) {
	now := float64(time.Now().Unix())
	votedKey := KeyPostVotedZSetPrefix + postID
	communityKey := KeyCommunityPostSetPrefix + communityName
	postInfo := map[string]interface{}{
		"title":    title,
		"summary":  summary,
		"post:id":  postID,
		"user:id":  userID,
		"time":     now,
		"votes":    0,
		"comments": 0,
	}
	// 事务操作
	pipeline := Client.TxPipeline()
	pipeline.ZAdd(votedKey, redis.Z{ // 作者默认投赞成票
		Score:  0,
		Member: userID,
	})
	pipeline.Expire(votedKey, time.Second*OneWeekInSeconds) // 一周时间
	pipeline.HMSet(KeyPostInfoHashPrefix+postID, postInfo)
	pipeline.ZAdd(KeyPostScoreZSet, redis.Z{ // 添加到分数的ZSet
		Score:  0,
		Member: postID,
	})
	pipeline.ZAdd(KeyPostTimeZSet+postID, redis.Z{ // 添加到时间的ZSet
		Score:  now,
		Member: postID,
	})
	pipeline.SAdd(communityKey, postID) // 添加到对应版块
	_, err = pipeline.Exec()
	return
}

// GetPost 从key中分页取出帖子
func GetPost(order string, page int64) []map[string]string {
	key := KeyPostScoreZSet
	if order == "time" {
		key = KeyPostTimeZSet
	}
	start := (page - 1) * PostPerAge
	end := start + PostPerAge - 1
	ids := Client.ZRevRange(key, start, end).Val()
	postList := make([]map[string]string, 0, len(ids))
	for _, id := range ids {
		postData := Client.HGetAll(KeyPostInfoHashPrefix + id).Val()
		postData["id"] = id
		postList = append(postList, postData)
	}
	return postList
}

// GetCommunityPost 分社区根据发帖时间或者分数取出分页的帖子
func GetCommunityPost(communityName, orderKey string, page int64) []map[string]string {
	key := orderKey + communityName // 创建缓存键

	if Client.Exists(key).Val() < 1 {
		Client.ZInterStore(key, redis.ZStore{
			Aggregate: "MAX",
		}, KeyCommunityPostSetPrefix+communityName, orderKey)
		Client.Expire(key, 60*time.Second)
	}
	return GetPost(key, page)
}

// Reddit Hot rank algorithms
// from https://github.com/reddit-archive/reddit/blob/master/r2/r2/lib/db/_sorts.pyx
func Hot(ups, downs int, date time.Time) float64 {
	s := float64(ups - downs)
	order := math.Log10(math.Max(math.Abs(s), 1))
	var sign float64
	if s > 0 {
		sign = 1
	} else if s == 0 {
		sign = 0
	} else {
		sign = -1
	}
	seconds := float64(date.Second() - 1577808000)
	return math.Round(sign*order + seconds/43200)
}
