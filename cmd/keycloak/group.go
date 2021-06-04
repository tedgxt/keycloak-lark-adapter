package keycloak

import (
	"encoding/json"
	"errors"
	"fmt"
	"keycloak-lark-adapter/cmd/lark"
	"keycloak-lark-adapter/internal/config"
	"keycloak-lark-adapter/internal/http"
	"keycloak-lark-adapter/internal/model/keycloak"
	lm "keycloak-lark-adapter/internal/model/lark"
	"keycloak-lark-adapter/pkg/utils"
	http2 "net/http"
	"strings"
)

func processDepMsgWithType(msg *lm.ContactDepMsg) error {
	token, err := getAppToken()
	if err != nil {
		return err
	}

	eventType := msg.Header.EventType
	switch eventType {
	case eventTypeDepartmentCreate:
		err = groupCreate(token, msg)
		if err != nil {
			logger.Errorf("process department create msg failed, error: %v", err.Error())
			return err
		}
	case eventTypeDepartmentUpdate:
		err = groupUpdate(token, msg)
		if err != nil {
			logger.Errorf("process department update msg failed, error: %v", err.Error())
			return err
		}
	case eventTypeDepartmentDelete:
		err = groupDelete(token, msg)
		if err != nil {
			logger.Errorf("process department delete msg failed, error: %v", err.Error())
			return err
		}

	default:
		errMsg := fmt.Sprintf("unsupport department event type: %v", eventType)
		logger.Errorf(errMsg)
		return errors.New(errMsg)
	}

	return nil
}

func groupCreate(token string, msg *lm.ContactDepMsg) (err error) {
	groupObj := msg.Event.Object

	lkParentDepId := groupObj.ParentDepartmentID
	if lkParentDepId == lark.RootDepartmentId {
		err = groupCreateEngine(token, groupObj.Name)
		if err != nil {
			return err
		}
	} else {
		fullDepNameInLark, err := lark.GetFullDepName(lkParentDepId)
		if err != nil {
			return err
		}

		// get group id in keycloak by full department name
		kcGroupId, err := getGroupIdByName(token, fullDepNameInLark)
		if err != nil {
			return err
		}
		err = subGroupCreateEngine(token, groupObj.Name, kcGroupId)
		if err != nil {
			return err
		}
	}

	return nil
}

func subGroupCreateEngine(token, groupName, parentGroupId string) error {
	logger.Debugf("creating sub group %v, parent group id: %v", groupName, parentGroupId)

	m := make(map[string]string)
	m["name"] = groupName

	resp, err := http.Client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", token).
		SetBody(m).
		Post(config.Host + "/auth/admin/realms/" + config.Realm + "/groups/" + parentGroupId + "/children")
	if err != nil {
		logger.Errorf("create sub group %v failed, error: %v", groupName, err.Error())
		return err
	}
	if !utils.IsSuccessResponse(resp.StatusCode(), http2.StatusConflict) {
		errMsg := fmt.Sprintf("create sub group %v response failed, code: %v, error msg: %v", groupName, resp.StatusCode(), string(resp.Body()))
		logger.Errorf(errMsg)

		return errors.New(errMsg)
	}
	return nil
}

func groupCreateEngine(token, groupName string) error {
	logger.Debugf("creating first class group %v", groupName)

	m := make(map[string]string)
	m["name"] = groupName

	resp, err := http.Client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", token).
		SetBody(m).
		Post(config.Host + "/auth/admin/realms/" + config.Realm + "/groups")
	if err != nil {
		logger.Errorf("create first class group  %v failed, error: %v", groupName, err.Error())
		return err
	}
	if !utils.IsSuccessResponse(resp.StatusCode(), http2.StatusConflict) {
		errMsg := fmt.Sprintf("create first class group %v failed, code: %v, error msg: %v", groupName, resp.StatusCode(), string(resp.Body()))
		logger.Errorf(errMsg)

		return errors.New(errMsg)
	}
	return nil
}

func groupDelete(token string, msg *lm.ContactDepMsg) error {
	groupObj := msg.Event.Object

	fullGroupNameInLark, err := lark.GetFullDepName(groupObj.OpenDepartmentID)
	if err != nil {
		return err
	}

	// get group id in keycloak by name
	groupId, err := getGroupIdByName(token, fullGroupNameInLark)
	if err != nil {
		// todo group 找不到时不报错
		return err
	}

	err = groupDeleteEngine(token, groupId)
	if err != nil {
		return err
	}

	return nil
}

func groupDeleteEngine(token, groupId string) error {
	logger.Debugf("deleting group, id: %v", groupId)

	resp, err := http.Client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", token).
		Delete(config.Host + "/auth/admin/realms/" + config.Realm + "/groups/" + groupId)
	if err != nil {
		logger.Errorf("delete group %v failed, error: %v", groupId, err.Error())
		return err
	}
	if !utils.IsSuccessResponse(resp.StatusCode()) {
		errMsg := fmt.Sprintf("delete group %v failed, code: %v, error msg: %v", groupId, resp.StatusCode(), string(resp.Body()))
		logger.Errorf(errMsg)

		return errors.New(errMsg)
	}
	return nil
}

func groupUpdate(token string, msg *lm.ContactDepMsg) error {
	depObj := msg.Event.Object
	depOldObj := msg.Event.OldObject

	// 修改group name
	if depOldObj.Name != "" {
		logger.Infof("updating group name(lark) %v to %v", depOldObj.Name, depObj.Name)
		newFullDepName, err := lark.GetFullDepName(depObj.OpenDepartmentID)
		if err != nil {
			return err
		}

		// 获取原部门的完整路径
		idx := strings.LastIndex(newFullDepName, depObj.Name)
		oldFullDepName := newFullDepName[:idx] + depOldObj.Name

		group, err := getGroup(token, oldFullDepName)
		if err != nil {
			return err
		}

		group.Name = depObj.Name
		if err = groupNameUpdateEngine(token, group); err != nil {
			return err
		}
	}

	// 修改group的层级
	if depOldObj.ParentDepartmentID != "" {
		logger.Infof("updating group id %v (lark) parent  id %v to %v", depObj.OpenDepartmentID, depOldObj.ParentDepartmentID, depObj.ParentDepartmentID)

		// 获取飞书中部门变更前的上级部门路径
		oldParentFullDepName, err := lark.GetFullDepName(depOldObj.ParentDepartmentID)
		if err != nil {
			return err
		}
		// 拼接处飞书中部门变更前，该部门的路径，并获取其在Keycloak中的group id
		oldFullDepName := oldParentFullDepName + "/" + depObj.Name
		groupId, err := getGroupIdByName(token, oldFullDepName)
		if err != nil {
			return err
		}

		var newParentId string
		// 通过飞书中新上级部门的id，获取上级部门在Keycloak中的id。
		// 根据飞书中新的上级部门parent_department_id是否为"0"，调用Keycloak的API是不一样的。
		if depObj.ParentDepartmentID != "0" {
			parentFullDepName, err := lark.GetFullDepName(depObj.ParentDepartmentID)
			if err != nil {
				return err
			}
			newParentId, err = getGroupIdByName(token, parentFullDepName)
			if err != nil {
				return err
			}
		}

		if err = groupParentUpdateEngine(token, groupId, newParentId); err != nil {
			return err
		}
		return nil
	}

	return nil
}

func groupParentUpdateEngine(token, groupId, newParentId string) (err error) {
	if newParentId == "" {
		resp, err := http.Client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Authorization", token).
			SetBody(map[string]string{"id": groupId}).
			Post(config.Host + "/auth/admin/realms/" + config.Realm + "/groups")
		if err != nil {
			logger.Errorf("update group %v to top level in keycloak failed, error: %v", groupId, err.Error())
			return err
		}
		if !utils.IsSuccessResponse(resp.StatusCode()) {
			errMsg := fmt.Sprintf("update group %v to top level in keycloak failed, response code: %v, response bdoy: %v", groupId, resp.StatusCode(), string(resp.Body()))
			logger.Errorf(errMsg)
			return errors.New(errMsg)
		}
	} else {
		resp, err := http.Client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Authorization", token).
			SetBody(map[string]string{"id": groupId}).
			Post(config.Host + "/auth/admin/realms/" + config.Realm + "/groups/" + newParentId + "/children")
		if err != nil {
			logger.Errorf("update group %v parent id to %v in keycloak failed, error: %v", groupId, newParentId, err.Error())
			return err
		}
		if !utils.IsSuccessResponse(resp.StatusCode()) {
			errMsg := fmt.Sprintf("update group %v parent to %v in keycloak failed, response code: %v, response bdoy: %v", groupId, newParentId, resp.StatusCode(), string(resp.Body()))
			logger.Errorf(errMsg)
			return errors.New(errMsg)
		}
	}

	return nil
}

func groupNameUpdateEngine(token string, group *keycloak.GroupInfo) (err error) {
	resp, err := http.Client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", token).
		SetBody(group).
		Put(config.Host + "/auth/admin/realms/" + config.Realm + "/groups/" + group.ID)
	if err != nil {
		logger.Errorf("update group %v in keycloak failed, error: %v", group.ID, err.Error())
		return err
	}
	if !utils.IsSuccessResponse(resp.StatusCode()) {
		errMsg := fmt.Sprintf("update group %v in keycloak failed, response code: %v, response bdoy: %v", group.ID, resp.StatusCode(), string(resp.Body()))
		logger.Errorf(errMsg)
		return errors.New(errMsg)
	}

	return nil
}

func getGroups(token string) (groups []*keycloak.GroupInfo, err error) {
	resp, err := http.Client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", token).
		Get(config.Host + "/auth/admin/realms/" + config.Realm + "/groups")
	if err != nil {
		logger.Errorf("get groups from keycloak failed, error: %v", err.Error())
		return nil, err
	}
	if !utils.IsSuccessResponse(resp.StatusCode()) {
		errMsg := fmt.Sprintf("get groups from keycloak failed, response code: %v, response bdoy: %v", resp.StatusCode(), string(resp.Body()))
		logger.Errorf(errMsg)
		return nil, errors.New(errMsg)
	}

	groups = []*keycloak.GroupInfo{}
	err = json.Unmarshal(resp.Body(), &groups)
	if err != nil {
		logger.Errorf("unmarshal groups failed, error: %v", err.Error())
		return nil, err
	}

	return groups, nil
}

func getGroup(token, fullGroupNameInLark string) (group *keycloak.GroupInfo, err error) {
	groups, err := getGroups(token)
	if err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		logger.Errorf("cannot find groups in keycloak")
		return nil, errors.New("cannot find groups in keycloak")
	}

	var f func(groups []*keycloak.GroupInfo) *keycloak.GroupInfo
	f = func(groups []*keycloak.GroupInfo) *keycloak.GroupInfo {
		for _, item := range groups {
			if item.Path == fullGroupNameInLark {
				return item
			}
			if item.SubGroups != nil && len(item.SubGroups) > 0 {
				group = f(item.SubGroups)
			}
			if group != nil {
				return group
			}
		}
		return nil
	}

	group = f(groups)
	if group == nil {
		errMsg := fmt.Sprintf("cannot find group %v from keycloak", fullGroupNameInLark)
		logger.Errorf(errMsg)
		return nil, errors.New(errMsg)
	}

	return group, nil
}

func getGroupIdByName(token, fullGroupNameInLark string) (groupId string, err error) {
	groups, err := getGroups(token)
	if err != nil {
		return "", err
	}
	if len(groups) == 0 {
		return "", errors.New("cannot find groups in keycloak")
	}

	var f func(groups []*keycloak.GroupInfo) string
	f = func(groups []*keycloak.GroupInfo) string {
		for _, group := range groups {
			if group.Path == fullGroupNameInLark {
				return group.ID
			}
			if group.SubGroups != nil && len(group.SubGroups) > 0 {
				groupId = f(group.SubGroups)
			}
			if groupId != "" {
				return groupId
			}
		}
		return ""
	}
	groupId = f(groups)
	if groupId == "" {
		return "", fmt.Errorf("cannot find group %v in keycloak", fullGroupNameInLark)
	}
	return groupId, nil
}

func getGroupIdInKeycloak(token string, userObj *lm.UserObject) (groupId string, err error) {
	// 从飞书获取到department信息，格式为"/Dev/HZ Dev/Ops & QA & LBware/Ops"
	fullGroupNameInLark, err := lark.GetFullDepName(userObj.DepartmentIDs[0])
	if err != nil {
		return "", err
	}
	if len(fullGroupNameInLark) == 0 {
		return "", nil
	}

	// get keycloak group id by name
	groupId, err = getGroupIdByName(token, fullGroupNameInLark)
	if err != nil {
		return "", err
	}

	return groupId, nil

}
