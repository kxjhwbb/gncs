package main

import (
	"github.com/dgrijalva/jwt-go"
	"gopkg.in/gin-gonic/gin.v1"
	"net/http"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"time"
	"math/rand"
	"strconv"

)


//产生随机字符串
func GetRandomString(l int) string{
	str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	bytes := []byte(str)
	result := []byte{}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < l; i++ {
		result = append(result, bytes[r.Intn(len(bytes))])
	}
	return string(result)
}

//红包结构
type Packet struct{
	Uid int
	Message string
	Type int
	Count int
	Money float64
}

//写入红包数据表，产出id + pass
func (p *Packet) Create()(id int64,pass string,err error){

	pass = GetRandomString(8) //随机密码
	rs, err :=db.Exec("INSERT INTO packet(uid,message,`type`,`count`,money,createtime,pass) VALUE (?,?,?,?,?,?,?)",
		p.Uid,p.Message,p.Type,p.Count,p.Money,time.Now().Unix(),pass)
	if err != nil{
		log.Fatalln(err)
	}
	id, err = rs.LastInsertId()
	if err != nil {
		log.Fatalln(err)
	}
	return
}


var db *sql.DB

func main() {
	var err error
	db, err = sql.Open("mysql", "gncs:T04*1CuPfVSUa8@tcp(holarholar.mysql.rds.aliyuncs.com:3200)/gncs?parseTime=true")
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(20)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatalln(err)
	}

	router := gin.Default()

	//首页
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK,"666")
	})



	//测试jwt登录
	router.GET("/login", func(c *gin.Context) {
		j := JWT{
			[]byte("gncs"),
		}
		claims := CustomClaims{
			1,
			"kxjhwbb",
			"702048@qq.com",
			jwt.StandardClaims{
				ExpiresAt: 15000, //time.Now().Add(24 * time.Hour).Unix()
				Issuer: "gncs",
			},
		}

		token, err := j.CreateToken(claims)
		if err != nil {
			c.String(http.StatusOK, err.Error())
			c.Abort()
		}
		//c.String(http.StatusOK, token+"---------------<br>")
		res, err := j.ParseToken(token)
		if err != nil {
			if err == TokenExpired {
				newToken, err := j.RefreshToken(token)
				if err != nil {
					c.String(http.StatusOK, err.Error())
				} else {
					c.JSON(http.StatusOK, gin.H{
						"token":newToken,
					})
				}
			} else {
				c.String(http.StatusOK, err.Error())
			}
		} else {
			c.JSON(http.StatusOK, res)
		}



	})


	//路由组
	authorize := router.Group("/", JWTAuth())
	{
		//展示当前登录用户
		authorize.GET("myprofile", func(c *gin.Context) {
			claims := c.MustGet("claims").(*CustomClaims)
			//fmt.Println(claims.ID)
			c.JSON(http.StatusOK, claims)
		})

		//发红包
		authorize.POST("createPacket", func(c *gin.Context) {
			claims := c.MustGet("claims").(*CustomClaims)
			//fmt.Println(claims.ID)
			//c.String(http.StatusOK, claims.Name)

			var err error

			Uid := claims.ID
			Message := c.Request.FormValue("msg")
			Type,err := strconv.Atoi(c.Request.FormValue("type"))
			Count,err := strconv.Atoi(c.Request.FormValue("count"))
			Money,err  := strconv.ParseFloat(c.Request.FormValue("money"),64)

			if err!=nil{
				c.String(http.StatusOK,err.Error())
				return
			}

			if Money<2 {
				c.JSON(http.StatusOK,gin.H{"errmsg":"红包金额不可低于2元!"})
				return
			}

			if Count>200 {
				c.JSON(http.StatusOK,gin.H{"errmsg":"红包个数不可超过200个!"})
				return
			}

			packet := Packet{
				Uid: Uid,
				Message: Message,
				Type: Type,
				Count: Count,
				Money: Money,
			}

			id,pass,err := packet.Create()

			if err != nil {
				log.Fatalln(err)
			}else{
				c.JSON(http.StatusOK,gin.H{
					"id":id,
					"pass":pass,
				})
			}

		})

	}




	router.Run(":8000")
}
