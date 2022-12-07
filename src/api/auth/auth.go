package auth

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/jtblin/go-ldap-client"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type AuthenticatedUser struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type AuthClaims struct {
	AuthenticatedUser
	jwt.StandardClaims
}

func ValidateBearerToken(auth_header string) (user *AuthenticatedUser, err error) {
	authz := strings.Split(auth_header, " ")
	if len(authz) != 2 || authz[0] != "Bearer" {
		err = fmt.Errorf("no authorization bearer token provided")
		return
	}
	user, err = VerifyToken(authz[1])
	if err != nil || user == nil {
		log.Errorf(err.Error())
	}
	return
}

func VerifyToken(token string) (user *AuthenticatedUser, err error) {
	parsed, err := jwt.ParseWithClaims(token, &AuthClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(viper.GetString("server.secret")), nil
	})
	if err != nil {
		return
	}
	if claims, ok := parsed.Claims.(*AuthClaims); ok && parsed.Valid {
		user = &claims.AuthenticatedUser
	}
	return
}

func GenerateToken(username string, password string) (token string, err error) {

	user, err := Authenticate(username, password)
	if err != nil || user == nil {
		return
	}
	j := jwt.NewWithClaims(jwt.SigningMethodHS256, AuthClaims{
		AuthenticatedUser: AuthenticatedUser{Name: user.Name, Email: user.Email},
		StandardClaims:    jwt.StandardClaims{NotBefore: time.Now().Unix()},
	})
	token, err = j.SignedString([]byte(viper.GetString("server.secret")))
	log.Infof("JWT for '%s' generated", user.Name)
	return
}

func Authenticate(username string, password string) (user *AuthenticatedUser, err error) {
	client := &ldap.LDAPClient{
		Base:         viper.GetString("ldap.base_dn"),
		Host:         viper.GetString("ldap.host"),
		Port:         viper.GetInt("ldap.port"),
		UseSSL:       viper.GetBool("ldap.use_ssl"),
		BindDN:       viper.GetString("ldap.bind_dn"),
		BindPassword: viper.GetString("ldap.bind_password"),
		UserFilter:   viper.GetString("ldap.user_filter"),
		GroupFilter:  viper.GetString("ldap.group_filter"),
		Attributes:   []string{"name", "mail", "distinguishedName"},
	}
	// It is the responsibility of the caller to close the connection
	defer client.Close()

	_, autheduser, err := client.Authenticate(username, password)

	if err != nil {
		m := err.Error()
		if strings.Contains(m, "Invalid Credentials") {
			log.Warnf("User '%s' not authenticated", username)
			return nil, nil
		}
		log.Errorf(m)
		return nil, err
	}
	user = &AuthenticatedUser{Name: autheduser["name"], Email: autheduser["mail"]}

	groups, err := client.GetGroupsOfUser(autheduser["distinguishedName"])
	if err != nil {
		log.Errorf("GetGroupsOfUser: %s\n", err.Error())
		return nil, err
	}

	for _, group := range groups {
		if group == viper.GetString("auth.required_group") {
			log.Infof("Authenticated '%s'", username)
			return user, nil
		}
	}

	log.Errorf("User '%s' not authorized", username)
	return nil, err
}
