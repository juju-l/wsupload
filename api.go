package v1

import (
	"errors"

	///
	"gopkg.in/yaml.v2"
	"gorm.io/gorm"

	"go-web/internal/repository"
	"go-web/internal/domain/dto"
	"go-web/internal/domain/entity"
	"go-web/internal/domain/inf"
	"go-web/util/ext"

	"net/http"; "io"; "io/ioutil"
	"strings"; "encoding/json"; "regexp"
	"time"

	"fmt"
)

var cfg *dto.ConfigDto
var repo inf.RepoInterface
var orm *gorm.DB

type ApiUsecase struct {}

func initCfg() {
	if cfg == nil {
		cfg = &dto.ConfigDto{}
		yaml.Unmarshal(
			ext.Must(ioutil.ReadFile("config/app_configs.yml")).([]byte),
		cfg)
		// fmt.Println(err)
	}
}

func (a *ApiUsecase) Auth(
			code string,
		) (*entity.User, error) {
	var err error
	user := &entity.User{}
	///
			if code != "debug" {
	if rsp, err := http.Get(fmt.Sprintf(cfg.Auth_code2Session, code)); err == nil {
		defer rsp.Body.Close(); body, _ := io.ReadAll(rsp.Body); json.Unmarshal(body, user)
		if user.OpenId == "" {
			return nil, errors.New(string(body))
		}
		//
	}
			} else {
			user.OpenId = "xxxxxxxxxxxxxxxxxxxxxxxxxxxx"
			}
	///
	if err = orm.Where("openid = ?", user.OpenId).First(user).Error;err != nil {
		orm.Save(user)
	}
	return user, err
}

func (a *ApiUsecase) UpContents(
			entId,openid,sig string,
		) (map[string]interface{}, error) {
	var err error
	cnt := make(map[string]any)
	if sig != "" {
					cnt["url"] = "https://xxx.vipex.cc:65443/stc/"+sig+"" //oss
	}
	// ---
	return cnt, err
}

func (a *ApiUsecase) GetAvatarUrl(
			openid,/*,*/url string,
		) (*entity.User, error) {
	var err error
	user := &entity.User{}
	orm.Where("openid = ?", openid).First(user)
	if url != "" {
			user.AvatarUrl = url
			user.IsAuthorization = true
			orm.Save(user)
	}
	return user, err
}

func checkTim(
			e string, o string, t *time.Time,
		) (*time.Time, error) {
	var err error
	err = orm.Raw(`
		SELECT 
			detail.timesp 
			FROM ent,detail,"order" 
		WHERE
			ent.entid = ? 
			AND ent.openid = ? 
			AND detail."id" = "order".ordid 
			AND unixepoch(detail.timesp) > unixepoch() 
			AND detail.is_valid = TRUE 
		ORDER BY detail.timesp DESC 
		  LIMIT 1
	`, e, o).Scan(t).Error
	return t, err
}

func (a *ApiUsecase) GetBusinessInfo(
			openid,/*,*/entid string,
		) (interface{}, error) {
	var err error
	result := struct { Name string `json:"name"`
			PeriodSrv string `json:"period_srv"`
	Sid string `json:"sid"` }{
	"暂未绑定", "0000-00-00", "暂未绑定",
	}
	ent := &entity.Ent{}
	if err = orm.
			Where("entid = ? AND openid = ?", entid, openid).
			First(ent).Error; err != nil {
		return result, nil
	}
	result.Name = ent.Name;result.Sid = ent.Sid
	timeSp := &time.Time{}
	timeSp, err = checkTim(entid, openid, timeSp)
	if err == nil {
	  	result.PeriodSrv = 
			timeSp.
			Format("2006-01-02")
	}
	return result, err
}

func (a *ApiUsecase) UpBusinessInfo(
			openid,/*,*/entId,name,num string,
		) (*entity.Ent, error) {
	var err error
	//
	sid := regexp.
		MustCompile(`(网络科技|技术服务)`).
		ReplaceAllString(strings.Replace(
			strings.Replace(name, "西安", "", -1),
			"有限公司",
			"",
		-1), "")
	//
	ent := &entity.Ent{
		entId,openid,name,num,"",true,sid,
	}
	orm.Save(ent)
	return ent, err
}

func (a *ApiUsecase) GetDetails(
			entId,openid,id string,
		) (map[string]interface{}, error) {
	var err error
	detail := make(map[string]interface{})
	err = orm.Table("detail").
		Select("detail").
		Where(
			//
			"entid = ? AND openid = ? AND id = ?",
			entId, openid, id,
		).
		Take(detail).
		Error
	if err == nil {
		json.Unmarshal(
			[]byte(detail["detail"].(string)), &detail,
		)
	}
	return detail, err
}

func savDetail(
			e ,o, id, d, s string,
		) (*entity.Detail, error) {
	var err error
	detail := &entity.Detail{
			EntId: e,
			OpenId: o,
			Id: id,
			Detail: d,
			Sid: s,
		}
	err = orm.Save(detail).
			Error
	return detail, err
}

func (a *ApiUsecase) GetOrder(
			openid,entid,name,num/*,sid*/ string,
		) (*[]entity.Detail, error) {
	detail := &[]entity.Detail{}
	ordid := &[]string{}
	if name != "" {
		insOrder := &entity.Order{ EntId: entid, OpenId: openid }; orm.Create(insOrder)
		savDetail(entid, openid, insOrder.OrdId, "{\"name\":\""+name+"\",\"num\":\""+num+"\"}", "")
		*ordid = append(*ordid, insOrder.OrdId)
	} else {
		/***/orm.Table("order").Select("ordid").Where("entid = ? AND openid = ?", entid, openid).Find(ordid)/***/
	}
	orm.Where(
	"entid = ? AND openid = ? AND id in (?)",
	entid, openid, *ordid).Find(detail)
	return detail, nil
}

func (a *ApiUsecase) GetCnts(
			entId,openid,sig string,
		) (map[string]interface{}, error) {
	var err error
			cnt := make(map[string]any)
	return cnt, err
}

func (a *ApiUsecase) GetRepair(
			openid,entid,description,details/*,sid*/ string,
		) (*[]entity.Detail, error) {
	detail := &[]entity.Detail{}
	repid := &[]string{}
	if description != "" {
		insOrder := &entity.Repair{ EntId: entid, OpenId: openid }; orm.Create(insOrder)
		savDetail(
			entid,
			openid,
			insOrder.Repid,
			"{\"description\":\""+description+"\",\"detail\":\""+details+"\"}",
			"",
		)
		*repid = append(*repid, insOrder.Repid)
	} else {
		orm.Table("repair").Select("repid").
			Where("entid = ? AND openid = ?", entid, openid).
			Find(repid)
	}
	orm.
	Where("entid = ? AND openid = ? AND id in (?)", entid, openid, *repid).
	Find(detail)
	return detail, nil
}

func initOrm() {
	if orm == nil {
			orm = ext.Must(repo.New()).
			(*repository.PsqlRepository).
			Orm
	}
}

// func f() {}

func init() {
	initCfg()
	repo = 
			&repository.
	PsqlRepository{}
	initOrm()
}