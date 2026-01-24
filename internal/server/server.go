package server

import (
	"api/config"
	"api/docs"
	"api/internal/service"
	"api/types"
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	swaggerFiles "github.com/swaggo/files"     // swagger embed files
	ginSwagger "github.com/swaggo/gin-swagger" // gin-swagger middleware
	"go.uber.org/fx"
)

const (
	sessionIDKey = "session_id"
	sessionName  = "simple-retro-session"
)

type Controller struct {
	service *service.Service
	config  *config.Config
	server  *http.Server
}

type ControllerParams struct {
	fx.In
	Service   *service.Service
	Config    *config.Config
	Lifecycle fx.Lifecycle
}

func New(p ControllerParams) *Controller {
	c := &Controller{
		service: p.Service,
		config:  p.Config,
	}

	p.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go c.Start()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return c.stop(ctx)
		},
	})

	return c
}

// CORSMiddleware to handle CORS headers
func CORSMiddleware(conf *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", fmt.Sprintf("http://%s:5173", conf.Server.Host))
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header(
			"Access-Control-Allow-Headers",
			"Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, User-Agent",
		)
		c.Header("Access-Control-Allow-Methods", "POST, HEAD, PATCH, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// Authenticate middleware to check for retrospective_id cookie
// it sets the retrospective_id in the gin context if present
func Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		retroIDcookie, err := c.Cookie("retrospective_id")
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not in any retrospective"})
			c.Abort()
			return
		}

		retroID, err := uuid.Parse(retroIDcookie)
		if err != nil {
			log.Printf("error parsing retrospective_id: %s", err.Error())
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not in any retrospective"})
			c.Abort()
			return
		}

		c.Set("retrospective_id", retroID)
	}
}

// EnsureSessionID middleware to ensure session ID exists
// it creates a new session ID if not present and saves it to the session store
// the sessionID is stored under the key "session_id" in gin context.
func EnsureSessionID() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)

		id := session.Get(sessionIDKey)
		if id == nil {
			newID := uuid.NewString()
			session.Set(sessionIDKey, newID)

			if err := session.Save(); err != nil {
				// Fail hard: session must exist
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "failed to create session",
				})
				return
			}
		}

		c.Set(sessionIDKey, session.Get(sessionIDKey).(string))
		c.Next()
	}

}

func newSessionStore(conf *config.Config) gin.HandlerFunc {
	store := cookie.NewStore([]byte(conf.SessionSecret))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
		Secure:   !conf.Development, // true in prod with HTTPS
	})

	return sessions.Sessions(sessionName, store)
}

// health godoc
//
//	@Summary	Show API health
//	@Tags		Healthcheck
//	@Produce	json
//	@Success	200	{object}	health	"API metrics"
//	@Failure	500	{string}	string	"Internal error"
//	@Router		/health [get]
func (ct *Controller) health(c *gin.Context) {
	health, err := getServiceHealth()
	if err != nil {
		log.Printf("error getting service health: %s", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error getting service health"})
		return
	}

	c.JSON(http.StatusOK, health)
}

// createRetrospective godoc
//
//	@Summary	Create Retrospective
//	@Tags		Retrospective
//	@Accept		json
//	@Produce	json
//	@Param		retrospective	body		types.RetrospectiveCreateRequest	true	"Create Retrospective"
//	@Success	200				{object}	types.Retrospective					"Retrospective Object"
//	@Failure	400				{string}	string								"Invalid input"
//	@Failure	500				{string}	string								"Internal error"
//	@Router		/retrospective [post]
func (ct *Controller) createRetrospective(c *gin.Context) {
	var input types.RetrospectiveCreateRequest
	if err := c.BindJSON(&input); err != nil {
		log.Printf("error parsing body content: %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body content"})
		return
	}

	if err := input.ValidateCreate(); err != nil {
		log.Printf("invalid input: %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	retrospective := types.Retrospective{
		Name:        input.Name,
		Description: input.Description,
		Questions:   []types.Question{},
	}

	err := ct.service.CreateRetrospective(c, &retrospective)
	if err != nil {
		log.Printf("error creating retrospective: %s", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, retrospective)
}

// getRetrospective godoc
//
//	@Summary	Get Retrospective by ID
//	@Tags		Retrospective
//	@Produce	json
//	@Param		id	path		string				true	"Retrospective ID"
//	@Success	200	{object}	types.Retrospective	"Retrospective Object"
//	@Failure	400	{string}	string				"Invalid input"
//	@Failure	404	{string}	string				"Not Found"
//	@Failure	500	{string}	string				"Internal error"
//	@Router		/retrospective/{id} [get]
func (ct *Controller) getRetrospective(c *gin.Context) {
	input := c.Param("id")
	id, err := uuid.Parse(input)
	if err != nil {
		log.Printf("error parsing path ID: %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	retro, err := ct.service.GetRetrospective(c, id)
	if err == sql.ErrNoRows {
		log.Printf("retrospective ID %s not found", id.String())
		c.JSON(http.StatusNotFound, gin.H{"error": "restrospective not found"})
		return
	}

	if err != nil {
		log.Printf("error getting retrospective: %s", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.SetSameSite(http.SameSiteNoneMode)
	c.SetCookie("retrospective_id", id.String(), 0, "/", "", true, false)
	c.JSON(http.StatusOK, retro)
}

// UpdateRetrospective godoc
//
//	@Summary	Update Retrospective by ID
//	@Tags		Retrospective
//	@Produce	json
//	@Param		id				path		string								true	"Retrospective ID"
//	@Param		retrospective	body		types.RetrospectiveCreateRequest	true	"Update Retrospective"
//	@Success	200				{object}	types.Retrospective					"Retrospective Object"
//	@Failure	400				{string}	string								"Invalid input"
//	@Failure	404				{string}	string								"Not Found"
//	@Failure	500				{string}	string								"Internal error"
//	@Router		/retrospective/{id} [patch]
func (ct *Controller) updateRetrospective(c *gin.Context) {
	input := c.Param("id")
	id, err := uuid.Parse(input)
	if err != nil {
		log.Printf("error parsing path ID: %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var inputRetro types.RetrospectiveCreateRequest
	if err := c.BindJSON(&inputRetro); err != nil {
		log.Printf("error parsing body content: %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body content"})
		return
	}

	if err := inputRetro.ValidateUpdate(); err != nil {
		log.Printf("invalid input: %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	retro := &types.Retrospective{
		ID:          id,
		Name:        inputRetro.Name,
		Description: inputRetro.Description,
		Questions:   []types.Question{},
	}

	err = ct.service.UpdateRetrospective(c, retro)

	if err == sql.ErrNoRows {
		log.Printf("retrospective ID %s not found", id.String())
		c.JSON(http.StatusNotFound, gin.H{"error": "restrospective not found"})
		return
	}

	if err != nil {
		log.Printf("error updating retrospective: %s", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, retro)
}

// deleteRetrospective godoc
//
//	@Summary	Delete Retrospective by ID
//	@Tags		Retrospective
//	@Produce	json
//	@Param		id	path		string				true	"Retrospective ID"
//	@Success	200	{object}	types.Retrospective	"Retrospective Object"
//	@Failure	400	{string}	string				"Invalid input"
//	@Failure	404	{string}	string				"Not Found"
//	@Failure	500	{string}	string				"Internal error"
//	@Router		/retrospective/{id} [delete]
func (ct *Controller) deleteRetrospective(c *gin.Context) {
	input := c.Param("id")
	id, err := uuid.Parse(input)
	if err != nil {
		log.Printf("error parsing path ID: %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	retro, err := ct.service.DeleteRetrospective(c, id)
	if err == sql.ErrNoRows {
		log.Printf("retrospective ID %s not found", id.String())
		c.JSON(http.StatusNotFound, gin.H{"error": "restrospective not found"})
		return
	}

	if err != nil {
		log.Printf("error deleting retrospective: %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	c.JSON(http.StatusOK, retro)
}

// createQuestion godoc
//
//	@Summary	Create Question
//	@Tags		Question
//	@Accept		json
//	@Produce	json
//	@Param		question	body		types.QuestionCreateRequest	true	"Create Question"
//	@Success	200			{object}	types.Question				"Retrospective Object"
//	@Failure	500			{string}	string						"Internal error"
//	@Router		/question [post]
func (ct *Controller) createQuestion(c *gin.Context) {
	var input types.QuestionCreateRequest
	if err := c.BindJSON(&input); err != nil {
		log.Printf("error parsing body content: %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body content"})
		return
	}

	if err := input.ValidateCreate(); err != nil {
		log.Printf("invalid input: %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	question := &types.Question{
		Text:    input.Text,
		Answers: []types.Answer{},
	}

	err := ct.service.CreateQuestion(c, question)
	if err != nil {
		if err.Error() == "FOREIGN KEY constraint failed" {
			log.Printf("error creating question: %s", err.Error())
			c.JSON(http.StatusBadRequest, gin.H{"error": "retrospective doesn't exist"})
			return
		}
		log.Printf("error creating question: %s", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, question)
}

// updateQuestion godoc
//
//	@Summary	Update Question by ID
//	@Tags		Question
//	@Produce	json
//	@Param		id				path		string						true	"Question ID"
//	@Param		retrospective	body		types.QuestionCreateRequest	true	"Update Question"
//	@Success	200				{object}	types.Retrospective			"Question Object"
//	@Failure	400				{string}	string						"Invalid input"
//	@Failure	404				{string}	string						"Not Found"
//	@Failure	500				{string}	string						"Internal error"
//	@Router		/question/{id} [patch]
func (ct *Controller) updateQuestion(c *gin.Context) {
	input := c.Param("id")
	id, err := uuid.Parse(input)
	if err != nil {
		log.Printf("error parsing path ID: %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var inputQuestion types.QuestionCreateRequest
	if err := c.BindJSON(&inputQuestion); err != nil {
		log.Printf("error parsing body content: %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body content"})
		return
	}

	if err := inputQuestion.ValidateCreate(); err != nil {
		log.Printf("invalid input: %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	question := &types.Question{
		ID:      id,
		Text:    inputQuestion.Text,
		Answers: []types.Answer{},
	}

	err = ct.service.UpdateQuestion(c, question)

	if err == sql.ErrNoRows {
		log.Printf("question ID %s not found", id.String())
		c.JSON(http.StatusNotFound, gin.H{"error": "question not found"})
		return
	}

	if err != nil {
		log.Printf("error updating question: %s", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, question)
}

// deleteQuestion godoc
//
//	@Summary	Delete Question by ID
//	@Tags		Question
//	@Produce	json
//	@Param		id	path		string			true	"Question ID"
//	@Success	200	{object}	types.Question	"Question Object"
//	@Failure	400	{string}	string			"Invalid input"
//	@Failure	404	{string}	string			"Not Found"
//	@Failure	500	{string}	string			"Internal error"
//	@Router		/question/{id} [delete]
func (ct *Controller) deleteQuestion(c *gin.Context) {
	input := c.Param("id")
	id, err := uuid.Parse(input)
	if err != nil {
		log.Printf("error parsing path ID: %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	question, err := ct.service.DeleteQuestion(c, id)
	if err == sql.ErrNoRows {
		log.Printf("question ID %s not found", id.String())
		c.JSON(http.StatusNotFound, gin.H{"error": "question not found"})
		return
	}

	if err != nil {
		log.Printf("error deleting question: %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	c.JSON(http.StatusOK, question)
}

// subscribeChanges godoc
//
//	@Summary	Subscribe to changes via web socket
//	@Tags		Websocket
//	@Accept		json
//	@Produce	json
//	@Param		id	path		string	true	"Repository ID"
//	@Failure	500	{string}	string	"Internal error"
//	@Router		/hello [get]
func (ct *Controller) subscribeChanges(c *gin.Context) {
	var err error
	retroIDparam := c.Param("id")
	if retroIDparam == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not in any retrospective"})
		return
	}

	retroID, err := uuid.Parse(retroIDparam)
	if err != nil {
		log.Printf("error parsing retrospective_id: %s", err.Error())
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not in any retrospective"})
		return
	}
	c.Set("retrospective_id", retroID)

	err = ct.service.SubscribeChanges(c, c.Writer, c.Request)
	if err != nil {
		errMessage := fmt.Errorf("error subscribing: %s", err.Error())
		log.Println(errMessage)
		c.JSON(http.StatusBadRequest, gin.H{"error": errMessage})
		return
	}

	c.JSON(http.StatusOK, "ok")
}

// createAnswer godoc
//
//	@Summary	Create Answer
//	@Tags		Answer
//	@Accept		json
//	@Produce	json
//	@Param		question	body		types.AnswerCreateRequest	true	"Create Answer"
//	@Success	200			{object}	types.Answer				"Retrospective Object"
//	@Failure	400			{string}	string						"Invalid input"
//	@Failure	500			{string}	string						"Internal error"
//	@Router		/answer [post]
func (ct *Controller) createAnswer(c *gin.Context) {
	var input *types.AnswerCreateRequest
	if err := c.BindJSON(&input); err != nil {
		log.Printf("error parsing body content: %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body content"})
		return
	}

	if err := input.ValidateCreate(); err != nil {
		log.Printf("invalid input: %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	answer := &types.Answer{
		QuestionID: input.QuestionID,
		Text:       input.Text,
	}

	err := ct.service.CreateAnswer(c, answer)
	if err != nil {
		log.Printf("error creating answer: %s", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, answer)
}

// updateAnswer godoc
//
//	@Summary	Update Answer
//	@Tags		Answer
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string						true	"Answer ID"
//	@Param		answer	body		types.AnswerCreateRequest	true	"Update Answer"
//	@Success	200		{object}	types.Answer				"Answer Object"
//	@Failure	400		{string}	string						"Invalid input"
//	@Failure	500		{string}	string						"Internal error"
//	@Router		/answer/{id} [patch]
func (ct *Controller) updateAnswer(c *gin.Context) {
	input := c.Param("id")
	id, err := uuid.Parse(input)
	if err != nil {
		log.Printf("error parsing path question ID: %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid question id"})
		return
	}

	var inputAnswer *types.AnswerCreateRequest
	if err := c.BindJSON(&inputAnswer); err != nil {
		log.Printf("error parsing body content: %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body content"})
		return
	}

	if err := inputAnswer.ValidateCreate(); err != nil {
		log.Printf("invalid input: %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	answer := &types.Answer{
		ID:         id,
		QuestionID: inputAnswer.QuestionID,
		Text:       inputAnswer.Text,
	}

	err = ct.service.UpdateAnswer(c, answer)
	if err == sql.ErrNoRows {
		log.Printf("answer ID %s not found", id.String())
		c.JSON(http.StatusNotFound, gin.H{"error": "answer not found"})
		return
	}

	if err != nil {
		log.Printf("error updating answer: %s", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, answer)
}

// deleteAnswer godoc
//
//	@Summary	Delete Answer
//	@Tags		Answer
//	@Accept		json
//	@Produce	json
//	@Param		id	path		string			true	"Answer ID"
//	@Success	200	{object}	types.Answer	"Answer Object"
//	@Failure	400	{string}	string			"Invalid input"
//	@Failure	500	{string}	string			"Internal error"
//	@Router		/answer/{id} [delete]
func (ct *Controller) deleteAnswer(c *gin.Context) {
	input := c.Param("id")
	id, err := uuid.Parse(input)
	if err != nil {
		log.Printf("error parsing path question ID: %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid question id"})
		return
	}

	answer := &types.Answer{
		ID: id,
	}
	err = ct.service.DeleteAnswer(c, answer)
	if err == sql.ErrNoRows {
		log.Printf("answer ID %s not found", id.String())
		c.JSON(http.StatusNotFound, gin.H{"error": "answer not found"})
		return
	}

	if err != nil {
		log.Printf("error deleting answer: %s", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, answer)
}

// getLimits godoc
//
//	@Summary	Get API limits
//	@Produce	json
//	@Success	200	{object}	types.ApiLimits	"API limits"
//	@Failure	500	{string}	string			"Internal error"
//	@Router		/limits [get]
func (ct *Controller) getLimits(c *gin.Context) {
	limits := ct.service.GetLimits(c)

	c.JSON(http.StatusOK, limits)
}

// voteAnswer godoc
//
//	@Summary	Vote Answer
//	@Tags		Answer
//	@Accept		json
//	@Produce	json
//	@Param		vote	body		types.AnswerVoteRequest	true	"Vote Answer"
//	@Success	200		{string}	string					"Vote recorded"
//	@Failure	400		{string}	string					"Invalid input"
//	@Failure	404		{string}	string					"Vote not found"
//	@Failure	409		{string}	string					"Vote already exists"
//	@Failure	500		{string}	string					"Internal error"
//	@Router		/answer/vote [post]
func (ct *Controller) voteAnswer(c *gin.Context) {
	var input types.AnswerVoteRequest
	if err := c.BindJSON(&input); err != nil {
		log.Printf("error parsing body content: %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body content"})
		return
	}

	if err := input.Validate(); err != nil {
		log.Printf("invalid input: %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sessionID := c.GetString(sessionIDKey)
	answer := &types.Answer{
		ID: input.AnswerID,
	}

	err := ct.service.VoteAnswer(c, answer, input.Action, sessionID)
	if err == nil {
		c.JSON(http.StatusOK, gin.H{"message": "vote recorded"})
		return
	}

	log.Printf("error voting answer: %s", err.Error())

	switch err {
	case service.ErrVoteAlreadyExists:
		c.JSON(http.StatusConflict, gin.H{"error": "vote already exists"})
		return
	case service.ErrVoteNotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": "vote not found"})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})

}

// @license.name	MIT
// @license.url	https://github.com/simple-retro/api/blob/master/LICENSE
func (ct *Controller) Start() {
	conf := ct.config

	// Swagger
	docs.SwaggerInfo.Title = conf.Name
	docs.SwaggerInfo.Description = "API service to Simple Retro project"
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Host = fmt.Sprintf("%s:%d", conf.Server.Host, conf.Server.Port)
	docs.SwaggerInfo.BasePath = "/api"
	docs.SwaggerInfo.Schemes = []string{"http", "https"}

	router := gin.Default()

	if conf.Server.WithCors {
		router.Use(CORSMiddleware(conf))
	}

	if conf.Development {
		router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	router.Use(newSessionStore(conf))
	router.Use(EnsureSessionID())

	api := router.Group("/api")
	api.GET("/health", ct.health)
	api.POST("/retrospective", ct.createRetrospective)
	api.GET("/retrospective/:id", ct.getRetrospective)
	api.PATCH("/retrospective/:id", ct.updateRetrospective)
	api.DELETE("/retrospective/:id", ct.deleteRetrospective)
	api.GET("/hello/:id", ct.subscribeChanges)
	api.GET("/limits", ct.getLimits)

	authorized := api.Group("/")
	authorized.Use(Authenticate())
	authorized.POST("/question", ct.createQuestion)
	authorized.PATCH("/question/:id", ct.updateQuestion)
	authorized.DELETE("/question/:id", ct.deleteQuestion)

	authorized.POST("/answer", ct.createAnswer)
	authorized.PATCH("/answer/:id", ct.updateAnswer)
	authorized.DELETE("/answer/:id", ct.deleteAnswer)
	authorized.POST("/answer/vote", ct.voteAnswer)

	addr := fmt.Sprintf(":%d", conf.Server.Port)
	ct.server = &http.Server{
		Addr:    addr,
		Handler: router,
	}

	log.Printf("starting server on %s", addr)
	if err := ct.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("error starting server: %s", err.Error())
	}
}

func (ct *Controller) stop(ctx context.Context) error {
	if ct.server != nil {
		log.Println("shutting down server...")
		return ct.server.Shutdown(ctx)
	}
	return nil
}
