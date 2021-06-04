package keycloak

import (
	"encoding/json"
	"errors"
	"fmt"
	"keycloak-lark-adapter/internal/config"
	"keycloak-lark-adapter/internal/http"
	"keycloak-lark-adapter/internal/model/keycloak"
	lm "keycloak-lark-adapter/internal/model/lark"
	"keycloak-lark-adapter/pkg/utils"
	http2 "net/http"
	"strings"
)

func processUserMsgWithType(msg *lm.ContactUserMsg) error {
	token, err := getAppToken()
	if err != nil {
		return err
	}

	eventType := msg.Header.EventType
	switch eventType {
	case eventTypeUserCreate:
		// 新用户加入飞书后，在未登录过keycloak时，keycloak中没有该用户，则不需要同步期间任何用户变动信息。
		// 当用户登录过keycloak时，会直接从飞书获取最新的用户信息填入keycloak中
		logger.Infof("process user create msg, currently using lark identity provider, do nothing")

	case eventTypeUserDelete:
		logger.Infof("received user %v delete msg, user will be deleted", msg.Event.Object.Email)

		err = userDelete(token, msg.Event.Object)
		if err != nil {
			logger.Errorf("process user delete msg failed, error: %v", err.Error())
			return err
		}
	case eventTypeUserUpdate:

		err = userUpdate(token, msg.Event.Object, msg.Event.OldObject)
		if err != nil {
			logger.Errorf("process user update msg failed, error: %v", err.Error())
			return err
		}
	default:
		errMsg := fmt.Sprintf("unsupport user event type: %v", eventType)
		logger.Errorf(errMsg)
		return errors.New(errMsg)
	}

	return nil
}

func userUpdate(token string, userObj, userOldObj *lm.UserObject) error {
	// 1. 激活事件，do nothing
	if userOldObj.Status != nil && !userOldObj.Status.IsActivated && userObj.Status.IsActivated {
		logger.Infof("received user %v or %v activate msg, do nothing", userObj.Name, userObj.Email)
		return nil
	}

	// 2. 修改email事件，直接删除keycloak中的老的email对应账号，并使用新的email创建一个新的用户
	if len(userOldObj.Email) > 0 {
		logger.Warnf("email changed from %v to %v, user will be deleted", userOldObj.Email, userObj.Email)

		userInKeycloak, err := getUserDetail(token, userOldObj.Email)
		if err != nil {
			logger.Errorf("get user %v id failed, error: %v", userOldObj.Email, err.Error())
			return err
		}
		if userInKeycloak == nil {
			return fmt.Errorf("cannot find user %v in keycloak", userOldObj.Email)
		}

		err = deleteUserEngine(token, userInKeycloak.Id)
		if err != nil {
			logger.Errorf("email %v changed to %v, delete user failed, error: %v", userOldObj.Email, userObj.Email, err.Error())
			return err
		}

		userCreate := genUser4Create(userObj)
		logger.Infof("trying to create user %v in keycloak", userObj.Email)
		err = createUser(token, userCreate)
		if err != nil {
			return err
		}
		userCreatedInKeycloak, err := getUserDetail(token, userCreate.Email)
		if err != nil {
			return err
		}

		// todo: check if no need to assign group
		logger.Infof("trying to assign group %v to user %v",
			userObj.DepartmentIDs[0], userObj.Email)
		err = assignGroup2User(token, userCreatedInKeycloak.Id, userObj)
		if err != nil {
			return err
		}
		return nil
	}

	// 3. 员工修改部门信息，若员工的email为空，报错。若员工在keycloak中不存在，进行创建。
	if len(userOldObj.DepartmentIDs) >= 0 {
		if len(userObj.Email) == 0 {
			errMsg := fmt.Sprintf("assign %v's department before assign email", userObj.Name)
			logger.Errorf(errMsg)
			return errors.New(errMsg)
		}

		logger.Infof("user %v changes department to %v", userObj.Email, userObj.DepartmentIDs)

		userInKeycloak, err := getUserDetail(token, userObj.Email)
		if err != nil {
			return err
		}
		if userInKeycloak == nil {
			logger.Infof("cannot find user %v in keycloak, trying to create", userObj.Email)

			userCreate := genUser4Create(userObj)
			err = createUser(token, userCreate)
			if err != nil {
				return err
			}
			userInKeycloak, err = getUserDetail(token, userObj.Email)
			if err != nil {
				return err
			}
		}

		// todo: check if no need to assign group
		logger.Infof("Trying to assign group %v to user %v",
			userObj.DepartmentIDs[0], userObj.Email)
		err = assignGroup2User(token, userInKeycloak.Id, userObj)
		if err != nil {
			return err
		}
	}

	// 4. 修改了员工信息，更新员工信息，并更新员工的部门信息
	// todo check the necessary to update user
	userNew, err := genUser4Update(token, userObj, userOldObj)
	if err != nil {
		logger.Errorf("generate user to update failed, error: %v", err.Error())
		return err
	}
	err = updateUser(token, userNew.Id, userNew)
	if err != nil {
		logger.Errorf("update user failed, error: %v", err.Error())
		return err
	}

	return nil
}

func userDelete(token string, userObj *lm.UserObject) error {

	userInKeycloak, err := getUserDetail(token, userObj.Email)
	if err != nil {
		logger.Errorf("get user %v failed, error: %v", userObj.Email, err.Error())
		return err
	}
	if userInKeycloak == nil {
		logger.Infof("cannot find user %v in keycloak, skip delete action", userObj.Email)
		return nil
	}

	err = deleteUserEngine(token, userInKeycloak.Id)
	if err != nil {
		logger.Errorf("delete user failed, error: %v", err.Error())
		return err
	}
	return nil
}

func getUserDetail(token, userName string) (user *keycloak.User, err error) {
	userList, err := getUserList(token)
	if err != nil {
		logger.Errorf("get user list from keycloak failed, error: %v", err.Error())
		return
	}
	if len(userList) == 0 {
		logger.Errorf("get user list empty")
		return nil, errors.New("get user list empty from keycloak")
	}
	for _, userItem := range userList {
		if userName == userItem.Username {
			logger.Debugf("get user detail from keycloak: %v", userItem)
			return userItem, nil
		}
	}
	errMsg := fmt.Sprintf("cannot find user %v in keycloak", userName)
	logger.Errorf(errMsg)
	return nil, nil
}

func getUserList(token string) (userList []*keycloak.User, err error) {
	resp, err := http.Client.R().
		SetHeader("Authorization", token).
		SetQueryParam("max", "10000").
		Get(config.Host + "/auth/admin/realms/" + config.Realm + "/users")
	if err != nil {
		logger.Errorf("get user list from keycloak failed, error: %v", err.Error())
		return nil, err
	}
	if !utils.IsSuccessResponse(resp.StatusCode()) {
		errMsg := fmt.Sprintf("get user list from keycloak failed, error code: %v, response: %v", resp.StatusCode(), string(resp.Body()))
		logger.Errorf(errMsg)
		return nil, errors.New(errMsg)
	}
	err = json.Unmarshal(resp.Body(), &userList)
	if err != nil {
		logger.Errorf("unmarshal user list failed, error: %v", err)
		return
	}
	return
}

func updateUser(token, userId string, user *keycloak.User) error {
	logger.Debugf("updating user with body: %v", user)
	resp, err := http.Client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", token).
		SetBody(*user).
		Put(config.Host + "/auth/admin/realms/" + config.Realm + "/users/" + userId)
	if err != nil {
		logger.Errorf("update user failed, error: %v", err.Error())
		return err
	}
	if !utils.IsSuccessResponse(resp.StatusCode()) {
		errMsg := fmt.Sprintf("update user response failed, code: %v, error msg: %v", resp.StatusCode(), string(resp.Body()))
		logger.Errorf(errMsg)

		return errors.New(errMsg)
	}
	return nil
}

func deleteUserEngine(token, userId string) error {
	resp, err := http.Client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", token).
		Delete(config.Host + "/auth/admin/realms/" + config.Realm + "/users/" + userId)
	if err != nil {
		logger.Errorf("delete user in keycloak failed, error: %v", err.Error())
		return err
	}
	if !utils.IsSuccessResponse(resp.StatusCode(), http2.StatusNotFound) {
		errMsg := fmt.Sprintf("delete user response failed, code: %v, error msg: %v", resp.StatusCode(), string(resp.Body()))
		logger.Errorf(errMsg)

		return errors.New(errMsg)
	}

	return nil
}

func parseName(name string) (realName string, nickname string) {
	if strings.Contains(name, "(") {
		nickname = strings.Split(name, "(")[0]
		realName = strings.Split(name, "(")[1]
	} else if strings.Contains(name, "（") {
		nickname = strings.Split(name, "（")[0]
		realName = strings.Split(name, "（")[1]
	} else {
		// 不包含左括号，则认为只有真名
		realName = strings.TrimSpace(name)
		return
	}

	if strings.Contains(realName, ")") {
		realName = strings.Split(realName, ")")[0]
	} else if strings.Contains(realName, "）") {
		realName = strings.Split(realName, "）")[0]
	}
	return strings.TrimSpace(realName), strings.TrimSpace(nickname)

}

func createUser(token string, user *keycloak.User) error {
	resp, err := http.Client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", token).
		SetBody(user).
		Post(config.Host + "/auth/admin/realms/" + config.Realm + "/users")
	if err != nil {
		logger.Errorf("create user in keycloak failed, error: %v", err.Error())
		return err
	}
	if !utils.IsSuccessResponse(resp.StatusCode()) {
		errMsg := fmt.Sprintf("create user response failed, code: %v, error msg: %v", resp.StatusCode(), string(resp.Body()))
		logger.Errorf(errMsg)

		return errors.New(errMsg)
	}

	return nil
}

func genUser4Update(token string, userObj *lm.UserObject, userOldObj *lm.UserObject) (user *keycloak.User, err error) {
	user = &keycloak.User{}

	// keycloak using email as username
	username := userObj.Email

	// keycloak中更新user时attributes是覆盖的，attributes需要先get再set
	userOldInKeycloak, err := getUserDetail(token, username)
	if err != nil {
		return nil, err
	}
	if userOldInKeycloak == nil {
		return nil, fmt.Errorf("cannot find user %v in keycloak", username)
	}

	attrs := userOldInKeycloak.Attributes
	if attrs == nil {
		attrs = map[string]interface{}{}
	}
	if userOldObj.Mobile != "" || userObj.Mobile != "" {
		attrs[attributePhoneNumber] = userObj.Mobile
	}
	if userOldObj.Name != "" || userObj.Name != "" {
		realName, nickName := parseName(userObj.Name)
		attrs[attributeRealName] = realName
		attrs[attributeNickname] = nickName
		user.LastName = realName
		user.FirstName = realName
	}
	user.Attributes = attrs
	user.Id = userOldInKeycloak.Id

	user.Enabled = userOldInKeycloak.Enabled
	if userOldObj.Status != nil {
		// 更新后的user是冻结状态，需要在keycloak中禁用user
		if userObj.Status.IsFrozen {
			// disable user
			logger.Infof("preparing to disable user %v", userObj.Email)
			*user.Enabled = false
		} else {
			logger.Infof("preparing to enable user %v", userObj.Email)
			*user.Enabled = true
		}
	}

	return
}

func deleteUserGroup(token, userId string) error {
	resp, err := http.Client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", token).
		Get(config.Host + "/auth/admin/realms/" + config.Realm + "/users/" + userId + "/groups")
	if err != nil {
		logger.Errorf("get user %v groups failed, error: %v", userId, err.Error())
		return err
	}
	if !utils.IsSuccessResponse(resp.StatusCode()) {
		errMsg := fmt.Sprintf("get user %v groups failed, response code: %v, response bdoy: %v", userId, resp.StatusCode(), string(resp.Body()))
		logger.Errorf(errMsg)
		return errors.New(errMsg)
	}
	userGroups := []*keycloak.GroupInfo{}
	err = json.Unmarshal(resp.Body(), &userGroups)
	if err != nil {
		logger.Errorf("unmarshal user %v groups info failed, error: %v", userId, err.Error())
		return err
	}

	for _, ug := range userGroups {
		resp, err := http.Client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Authorization", token).
			Delete(config.Host + "/auth/admin/realms/" + config.Realm + "/users/" + userId + "/groups/" + ug.ID)
		if err != nil {
			logger.Errorf("delete user %v group %v failed, error: %v", userId, ug.ID, err.Error())
			return err
		}

		if !utils.IsSuccessResponse(resp.StatusCode(), http2.StatusNotFound) {
			errMsg := fmt.Sprintf("delete user %v group %v failed, code: %v, error msg: %v", userId, ug.ID, resp.StatusCode(), string(resp.Body()))
			logger.Errorf(errMsg)

			return errors.New(errMsg)
		}
	}

	return nil
}

// 飞书中只能给用户指定一个部门，而keycloak可以给用户指定多个group，需要删除keycloak中用户原有的group后再指定新的group。
// 飞书中当前展示的组织架构从公司下面的一级部门开始，keycloak中的group也是从一级部门开始。而飞书后台可以给用户指定部门为公司，
// 此时认为用户不属于任何一个部门，在keycloak中删除该用户的所有group信息。
func assignGroup2User(token, userId string, userObj *lm.UserObject) error {
	groupId, err := getGroupIdInKeycloak(token, userObj)
	if err != nil {
		return err
	}

	err = deleteUserGroup(token, userId)
	if err != nil {
		return err
	}
	if groupId == "" {
		logger.Infof("lark set user %v department to root, delete user's group in keycloak", userId)
		return nil
	}

	groupAssignment := &keycloak.GroupAssignment{
		GroupID: groupId,
		Realm:   config.Realm,
		UserID:  userId,
	}

	resp, err := http.Client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", token).
		SetBody(groupAssignment).
		Put(config.Host + "/auth/admin/realms/" + config.Realm + "/users/" + userId + "/groups/" + groupId)
	if err != nil {
		logger.Errorf("assign grouop %v to user %v failed, error: %v", groupId, userId, err.Error())
		return err
	}
	if !utils.IsSuccessResponse(resp.StatusCode()) {
		errMsg := fmt.Sprintf("assign group %v to user %v failed, response code: %v, response bdoy: %v", groupId, userId, resp.StatusCode(), string(resp.Body()))
		logger.Errorf(errMsg)

		return errors.New(errMsg)
	}

	return nil
}

func genUser4Create(userObj *lm.UserObject) (user *keycloak.User) {
	user = &keycloak.User{}
	// keycloak using email as username
	user.Username = userObj.Email
	user.Email = userObj.Email
	enable := true
	user.Enabled = &enable

	attrs := map[string]interface{}{}
	attrs[attributePhoneNumber] = userObj.Mobile

	realName, nickName := parseName(userObj.Name)
	attrs[attributeRealName] = realName
	attrs[attributeNickname] = nickName
	user.Attributes = attrs

	user.LastName = realName
	user.FirstName = realName

	return user
}
