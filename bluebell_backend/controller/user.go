package controller

import (
	"bluebell_backend/dao/mysql"
	"bluebell_backend/dao/redis"
	"bluebell_backend/models"
	"bluebell_backend/pkg/jwt"
	"bluebell_backend/utils"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func SignUpHandler(c *gin.Context) {
	// 拿到用户输入的注册信息 存储到 fo 结构体中
	var fo models.RegisterForm
	if err := c.ShouldBindJSON(&fo); err != nil {
		ResponseErrorWithMsg(c, CodeInvalidParams, "注册时候，出错，无法将用户输入信息存储到结构体中")
		return
	}
	// 查询用户是否存在
	err := mysql.Register(&models.User{
		UserName: fo.UserName,
		Password: fo.Password,
	})
	if errors.Is(err, mysql.ErrorUserExit) {
		ResponseError(c, CodeUserExist)
		return
	}
	if err != nil {
		zap.L().Error("mysql.Register() failed", zap.Error(err))
		ResponseError(c, CodeServerBusy)
		return
	}
	Rd := &ResponseData{
		Code:    CodeSuccess,
		Message: CodeSuccess.Msg(),
		Data:    fo,
	}
	ResponseSuccess(c, Rd)
}

// 登录页面
func LoginHandler(c *gin.Context) {
	var u models.User
	if err := c.ShouldBindJSON(&u); err != nil {
		zap.L().Error("用户登录时，信息绑定到结构体失败", zap.Error(err))
		ResponseErrorWithMsg(c, CodeInvalidParams, err.Error())
		return
	}
	if err := mysql.Login(&u); err != nil {
		zap.L().Error("mysql.Login(&u) failed", zap.Error(err))
		ResponseErrorWithMsg(c, CodeInvalidParams, err.Error())
		return
	}
	if flag := utils.CaptchaVerify(u.CaptchaId); flag == false {
		zap.L().Error("验证码输入错误")
		ResponseErrorWithMsg(c, CodeParam, "验证码错误")
		return
	}
	redis.Client.Set("bluebell:userID:", u.UserID, 12*time.Hour)
	//生成Token
	aToken, rToken, _ := jwt.GenToken(u.UserID)
	ResponseSuccess(c, gin.H{
		"accessToken":  aToken,
		"refreshToken": rToken,
		"userID":       u.UserID,
		"username":     u.UserName,
		"code":         http.StatusOK,
	})
}

func RefreshTokenHandler(c *gin.Context) {
	rt := c.Query("refresh_token")
	// 客户端携带Token有三种方式 1.放在请求头 2.放在请求体 3.放在URI
	// 这里假设Token放在Header的Authorization中，并使用Bearer开头
	// 这里的具体实现方式要依据你的实际业务情况决定
	authHeader := c.Request.Header.Get("Authorization")
	if authHeader == "" {
		ResponseErrorWithMsg(c, CodeInvalidToken, "请求头缺少Auth Token")
		c.Abort()
		return
	}
	// 按空格分割
	parts := strings.SplitN(authHeader, " ", 2)
	if !(len(parts) == 2 && parts[0] == "Bearer") {
		ResponseErrorWithMsg(c, CodeInvalidToken, "Token格式不对")
		c.Abort()
		return
	}
	aToken, rToken, err := jwt.RefreshToken(parts[1], rt)
	fmt.Println(err)
	c.JSON(http.StatusOK, gin.H{
		"access_token":  aToken,
		"refresh_token": rToken,
	})
}

// 展示用户信息
func ListUserInformation(c *gin.Context) {
	userid, err := redis.Client.Get("bluebell:userID:").Result()
	fmt.Println(userid)
	if err != nil {
		zap.L().Error("从redis中拿取数据失败，ListUserInformaiton，请重新尝试")
		ResponseErrorWithMsg(c, CodeInvalidParams, "从redis中拿取数据失败，ListUserInformaiton，请重新尝试")
		return
	}
	var u models.User
	if err1 := mysql.DB.Where("user_id=?", userid).First(&u).Error; err1 != nil {
		ResponseErrorWithMsg(c, CodeError, "从数据库中拿取相应的用户信息失败")
		return
	}
	ResponseSuccess(c, u)
}
