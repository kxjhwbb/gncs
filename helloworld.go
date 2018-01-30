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

	"fmt"
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
	Pid int
	Uid int
	Message string
	Type int
	Count int
	Money float64
	Pass string
}

//红包领取信息结构
type Got struct{
	Uid int
	Pid int
	Money float64
	Subtype int
}

//读取红包数据
func (p *Packet) GetPacket() (packet Packet, err error) {
	err = db.QueryRow("SELECT pid,money,count FROM packet WHERE pid=? and pass=?", p.Pid,p.Pass).Scan(
		&packet.Pid, &packet.Money, &packet.Count,
	)
	return
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

//红包信息更新
func (p *Packet) ModPacket() (ra int64, err error) {

	fmt.Println(p.Money);
	fmt.Println(p.Pid)

	rs, err := db.Exec("UPDATE packet SET money=money-?,gotmoney=gotmoney+?,`count`=`count`-1,gotcount=gotcount+1 WHERE pid=?",p.Money , p.Money , p.Pid)
	if err != nil {
		log.Fatalln(err)
		return
	}
	ra, err = rs.RowsAffected()
	if err != nil {
		log.Fatalln(err)
		return
	}
	return
}


//写入抢红包数据表
func (g *Got) Create()(id int64,err error){

	rs, err :=db.Exec("INSERT INTO got(uid,pid,money,subtype,createtime) VALUE (?,?,?,?,?)",
		g.Uid,g.Pid,g.Money,g.Subtype,time.Now().Unix())
	if err != nil{
		log.Fatalln(err)
	}
	id, err = rs.RowsAffected()
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
		//展示当前登录用户信息
		authorize.GET("myprofile", func(c *gin.Context) {
			claims := c.MustGet("claims").(*CustomClaims)
			//fmt.Println(claims.ID)
			c.JSON(http.StatusOK, claims)
		})

		//发红包
		authorize.POST("createPacket", func(c *gin.Context) {
			claims := c.MustGet("claims").(*CustomClaims)

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

		//抢红包
		authorize.POST("withdrawPacket", func(c *gin.Context) {
			claims := c.MustGet("claims").(*CustomClaims)
			Uid := claims.ID

			Pid,err := strconv.Atoi(c.Request.FormValue("pid"))
			Pass := c.Request.FormValue("pass")


			if err!=nil{
				c.String(http.StatusOK,err.Error())
				return
			}

			packet := Packet{
				Pid:Pid,
				Pass:Pass,
			}

			gprs,err := packet.GetPacket() //红包表查信息
			if err!=nil{
				c.JSON(200,gin.H{
					"errmsg":"红包不存在的",
				})
				return
			}

			if gprs.Count>0{
				//有红包

				Money := gprs.Money/2 //临时抢到的红包金额

				got := Got{
					Pid: Pid,
					Uid: Uid,
					Money: Money,
					Subtype: 1,
				}
				cgrs,err := got.Create() //红包领取表更新

				if err!=nil{
					c.String(http.StatusOK,err.Error())
					return
				}

				if cgrs >0 {

					packet = Packet{
						Pid:Pid,
						Money:Money,
					}

					mprs,err := packet.ModPacket() //红包表更新
					if err!=nil{
						c.String(http.StatusOK,err.Error())
						return
					}
					if(mprs>0){
						c.JSON(200,gin.H{
							"return":"success",
						})
					}

				}else{
					c.JSON(200,gin.H{
						"errmsg":"更新红包表异常!",
					})
				}

			}else{
				//无红包

				c.JSON(200,gin.H{
					"errmsg":"红包不存在或已抢完",
				})

			}



		})

		//提现

	}




	router.Run(":8000")
}
