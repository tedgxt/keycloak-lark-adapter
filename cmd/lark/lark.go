package lark

import (
	"encoding/json"
	"errors"
	"fmt"
	"keycloak-lark-adapter/internal/config"
	"keycloak-lark-adapter/internal/http"
	log "keycloak-lark-adapter/internal/logger"
	"keycloak-lark-adapter/internal/model/lark"
	"keycloak-lark-adapter/pkg/utils"

	"github.com/bitly/go-simplejson"
	"github.com/sirupsen/logrus"
)

const (
	RootDepartmentId = "0"
)

var (
	logger *logrus.Logger
)

func Init() {
	logger = log.Logger
}

func getAppToken() (token string, err error) {
	m := make(map[string]string)
	m["app_id"] = config.AppId
	m["app_secret"] = config.AppSecret

	resp, err := http.Client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(m).
		Post("https://open.feishu.cn/open-apis/auth/v3/app_access_token/internal/")
	if err != nil {
		logger.Errorf("get app token from lark failed, error: %v", err.Error())
		return "", err
	}
	if !utils.IsSuccessResponse(resp.StatusCode()) {
		errMsg := fmt.Sprintf("get app token from lark failed, response code: %v, response bdoy: %v", resp.StatusCode(), string(resp.Body()))
		logger.Errorf(errMsg)

		return "", errors.New(errMsg)
	}

	sj, err := simplejson.NewJson(resp.Body())
	if err != nil {
		logger.Errorf("get app token failed，error: %v", err.Error())
		return "", err
	}
	token = sj.Get("app_access_token").MustString()
	if token == "" {
		errMsg := fmt.Sprintf("get app token from lark failed, response code: %v, response bdoy: %v", resp.StatusCode(), string(resp.Body()))
		logger.Errorf(errMsg)

		return "", errors.New(errMsg)
	}

	return "Bearer " + token, nil
}

// GetFullDepName get full path of current department by department id
func GetFullDepName(depId string) (depName string, err error) {
	token, err := getAppToken()
	if err != nil {
		return "", err
	}
	if depId == RootDepartmentId {
		return "", nil
	}

	// 若depId不为0，则递归查找parent department
	for {
		depResp, err := GetDepInfo(token, depId)
		if err != nil {
			return "", err
		}

		depName = "/" + depResp.Data.Department.Name + depName
		depId = depResp.Data.Department.ParentDepartmentID
		if depResp.Data.Department.ParentDepartmentID != RootDepartmentId {
			continue
		}
		return depName, nil
	}
}

func GetDepInfo(token, depId string) (depResp *lark.DepartmentResponse, err error) {
	resp, err := http.Client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", token).
		Get("https://open.feishu.cn/open-apis/contact/v3/departments/" + depId)
	if err != nil {
		logger.Errorf("get department %v info from lark failed, error: %v", depId, err.Error())
		return nil, err
	}
	if !utils.IsSuccessResponse(resp.StatusCode()) {
		errMsg := fmt.Sprintf("get department info by id %v from lark failed, response code: %v, response bdoy: %v", depId, resp.StatusCode(), string(resp.Body()))
		logger.Errorf(errMsg)

		return nil, errors.New(errMsg)
	}

	depResp = new(lark.DepartmentResponse)
	err = json.Unmarshal(resp.Body(), depResp)
	if err != nil {
		logger.Errorf("unmarshal department info failed, error: %v", err)
		return
	}
	return depResp, nil
}
