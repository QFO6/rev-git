package controllers

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	revgit "github.com/QFO6/rev-git"
	gitgrpc "github.com/QFO6/rev-git/lib/gitgrpc"
	revmongo "github.com/QFO6/rev-mongo"
	utilsgo "github.com/QFO6/utils-go"

	"github.com/globalsign/mgo/bson"
	"github.com/revel/revel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type GitAPI struct {
	*revel.Controller
	revmongo.MgoController
}

func (c *GitAPI) CommitContent(modelName, id, commitHash string) revel.Result {
	if !CheckGitConfig() {
		return c.RenderJSON(map[string]string{
			"status":  "failed",
			"message": fmt.Sprintf("GitGrpcUrl, GitUrl, GitToken(or GitUser and GitPass) are mandatory in %s util value", revgit.GitUtilName),
		})
	}

	// #no sec G402
	config := &tls.Config{
		InsecureSkipVerify: true,
	}
	conn, err := grpc.DialContext(context.Background(), revgit.GitGrpcUrl,
		grpc.WithDefaultCallOptions(),
		grpc.WithTransportCredentials(credentials.NewTLS(config)))

	if err != nil {
		return c.RenderJSON(map[string]string{
			"status":  "failed",
			"message": "cannot connect to git server",
		})
	}

	defer conn.Close()

	cli := gitgrpc.NewGitServiceClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()

	relativePath := path.Join(revel.AppName)
	fileName := fmt.Sprintf("%s_%s_%s", revel.AppName, modelName, id)

	req := &gitgrpc.Request{
		RelativePath: relativePath,
		FileName:     fileName,
		CommitHash:   commitHash,
	}

	r, err := cli.ReadCommitContent(ctx, req)

	if err != nil {
		return c.RenderJSON(map[string]string{
			"status":  "failed",
			"message": err.Error(),
		})
	}
	status := r.GetStatus()

	if status != "success" {
		return c.RenderJSON(map[string]string{
			"status":  "failed",
			"message": r.GetMessage(),
		})
	}
	return c.RenderJSON(map[string]string{
		"status":  "success",
		"message": r.GetMessage(),
	})
}

// SaveToGit retrieve json data and save record to git if defined
func (c *GitAPI) Commit(modelName, id string) revel.Result {
	if !CheckGitConfig() {
		return c.RenderJSON(map[string]string{
			"status":  "failed",
			"message": fmt.Sprintf("GitGrpcUrl, GitUrl, GitToken(or GitUser and GitPass) are mandatory in %s util value", revgit.GitUtilName),
		})
	}

	var userName string
	var userEmail string
	if v, found := c.Session["Email"]; found {
		userEmail = v.(string)
	}

	if v, found := c.Session["UserName"]; found {
		userName = v.(string)
	}

	if strings.TrimSpace(userName) == "" {
		userName = strings.TrimSpace(c.Params.Get("user_name"))
	}

	if strings.TrimSpace(userName) == "" {
		userEmail = strings.TrimSpace(c.Params.Get("user_email"))
	}

	if strings.TrimSpace(userName) == "" || strings.TrimSpace(userEmail) == "" {
		return c.RenderJSON(map[string]string{"status": "failure", "message": "No user name or email provided"})
	}

	var jsonData map[string]interface{}
	err := c.Params.BindJSON(&jsonData)
	if err != nil {
		return c.RenderError(err)
	}
	jsonStr, _ := json.Marshal(jsonData)

	gitHash, err := commitToGit(modelName, id, string(jsonStr), userName, userEmail)
	if err != nil {
		return c.RenderJSON(map[string]string{"status": "failure", "message": fmt.Sprintf("%v", err)})
	}
	// send to gitGrpc service
	return c.RenderJSON(map[string]string{"status": "success", "message": gitHash})
}

func (c *GitAPI) History(modelName, id string) revel.Result {
	if !CheckGitConfig() {
		return c.RenderJSON(map[string]string{
			"status":  "failed",
			"message": fmt.Sprintf("GitGrpcUrl, GitUrl, GitToken(or GitUser and GitPass) are mandatory in %s util value", revgit.GitUtilName),
		})
	}

	// #nosec G402
	config := &tls.Config{
		InsecureSkipVerify: true,
	}
	conn, err := grpc.DialContext(context.Background(), revgit.GitGrpcUrl,
		grpc.WithDefaultCallOptions(),
		grpc.WithTransportCredentials(credentials.NewTLS(config)))

	if err != nil {
		return c.RenderJSON(map[string]string{
			"status":  "failed",
			"message": "cannot connect to git server",
		})
	}

	defer conn.Close()

	cli := gitgrpc.NewGitServiceClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()

	relativePath := path.Join(revel.AppName)
	fileName := fmt.Sprintf("%s_%s_%s", revel.AppName, modelName, id)

	req := &gitgrpc.Request{
		RelativePath: relativePath,
		FileName:     fileName,
	}

	r, err := cli.ReadFileHistory(ctx, req)

	if err != nil {
		// return c.RenderJSON(failureResp(err))
		return c.RenderJSON(map[string]string{
			"status":  "failed",
			"message": "cannot connect to git server",
		})
	}
	status := r.GetStatus()

	if status != "success" {
		return c.RenderJSON(map[string]string{
			"status":  "failed",
			"message": r.GetMessage(),
		})
	}
	return c.RenderJSON(map[string]string{"status": "success", "message": r.GetMessage()})
}

func commitToGit(modelName, id, jsonData, by string, mail string) (string, error) {
	// #nosec G402
	config := &tls.Config{
		InsecureSkipVerify: true,
	}
	conn, err := grpc.DialContext(context.Background(), revgit.GitGrpcUrl,
		grpc.WithDefaultCallOptions(),
		grpc.WithTransportCredentials(credentials.NewTLS(config)))

	if err != nil {
		return "", err
	}
	defer conn.Close()

	cli := gitgrpc.NewGitServiceClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()

	relativePath := path.Join(revel.AppName)
	fileName := fmt.Sprintf("%s_%s_%s", revel.AppName, modelName, id)
	committerName := by
	committerEmail := mail
	commitMessage := fmt.Sprintf("update %s", fileName)
	content := jsonData

	req := &gitgrpc.Request{
		RelativePath:   relativePath,
		FileName:       fileName,
		CommitterName:  committerName,
		CommitterEmail: committerEmail,
		CommitMessage:  commitMessage,
		Content:        content,
		GitUrl:         revgit.GitUrl,
		GitUsername:    revgit.GitUser,
		GitPassword:    revgit.GitPass,
		GitToken:       revgit.GitToken,
	}

	r, err := cli.SaveToGit(ctx, req)

	if err != nil {
		return "", err
	}
	status := r.GetStatus()
	if status != "success" {
		return "", fmt.Errorf("%s", r.GetMessage())
	}
	return r.GetMessage(), nil
}

func CheckGitConfig() bool {
	fmt.Printf("%s: \nGitGrpcUrl:%s\nGitUrl:%s\nGitToken:%s\n", revgit.GitUtilName, revgit.GitGrpcUrl, revgit.GitUrl, revgit.GitToken)
	return strings.TrimSpace(revgit.GitGrpcUrl) != "" &&
		strings.TrimSpace(revgit.GitUrl) != "" &&
		(strings.TrimSpace(revgit.GitToken) != "" ||
			(strings.TrimSpace(revgit.GitUser) != "" && strings.TrimSpace(revgit.GitPass) != ""))
}

// before check authToken
// checkToken
// Token defined in header named as AuthToken
// or in params with AuthToken as query name
func (c *GitAPI) CheckToken() revel.Result {
	// check request from same origin site
	if c.IsSameHostRefer() {
		return nil
	}

	// check identity session only if called from logged user in application
	if _, found := c.Session["Identity"]; found {
		return nil
	}

	// check the access token if called from a http request
	var tokenStr string
	if v, found := c.Session["AuthToken"]; found {
		tokenStr = v.(string)
	}

	if tokenStr == "" {
		tokenStr = c.Request.Header.Get("AuthToken")
	}

	// check header
	if tokenStr == "" {
		tokenStr = c.Params.Get("AuthToken")
	}

	// Using AccessToken as AuthToken
	util := new(revmongo.Utils)
	do := revmongo.New(c.MgoSession, util)
	do.Query = bson.M{"Name": "AccessToken"}
	err := do.GetByQ()
	if err != nil {
		return c.RenderJSON(map[string]string{"status": utilsgo.FAILURE, "error": "AccessToken not defined"})
	}

	if tokenStr == util.Value {
		return nil
	}

	c.Response.Status = http.StatusUnauthorized
	return c.RenderError(fmt.Errorf("401: Unauthorized"))
}

// isSameHostRefer check request.Referer host same as request.Host
func (c *GitAPI) IsSameHostRefer() bool {
	// current Host except port
	currentHost := strings.Split(c.Request.Host, ":")[0]
	referHost, _ := url.Parse(c.Request.Referer())
	return currentHost == referHost.Hostname()
}
