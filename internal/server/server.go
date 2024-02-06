package server

import (
	"api/config"
	"api/docs"
	"api/internal/service"
	"api/types"
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	swaggerFiles "github.com/swaggo/files"     // swagger embed files
	ginSwagger "github.com/swaggo/gin-swagger" // gin-swagger middleware
)

type controller struct {
	service *service.Service
}

func New(s *service.Service) *controller {
	return &controller{
		service: s,
	}
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "http://127.0.0.1:5173")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header(
			"Access-Control-Allow-Headers",
			"Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, User-Agent",
		)
		c.Header("Access-Control-Allow-Methods", "POST,HEAD,PATCH, OPTIONS, GET, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

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

// health godoc
//
//	@Summary	Show API health
//	@Tags Healthcheck
//	@Produce	json
//	@Success	200	{object}	health	"API metrics"
//	@Failure	500	{string}	string	"Internal error"
//	@Router		/health [get]
func (ct *controller) health(c *gin.Context) {
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
func (ct *controller) createRetrospective(c *gin.Context) {
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
func (ct *controller) getRetrospective(c *gin.Context) {
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
func (ct *controller) updateRetrospective(c *gin.Context) {
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
func (ct *controller) deleteRetrospective(c *gin.Context) {
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
func (ct *controller) createQuestion(c *gin.Context) {
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
		Text: input.Text,
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
func (ct *controller) updateQuestion(c *gin.Context) {
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
		ID:   id,
		Text: inputQuestion.Text,
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
func (ct *controller) deleteQuestion(c *gin.Context) {
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
func (ct *controller) subscribeChanges(c *gin.Context) {
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
func (ct *controller) createAnswer(c *gin.Context) {
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

	c.Set("question_id", input.QuestionID)
	answer := &types.Answer{
		Text: input.Text,
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
func (ct *controller) updateAnswer(c *gin.Context) {
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

	c.Set("question_id", inputAnswer.QuestionID)
	answer := &types.Answer{
		ID:   id,
		Text: inputAnswer.Text,
	}

	err = ct.service.UpdateAnswer(c, answer)
	if err == sql.ErrNoRows {
	}

	if err != nil {
		log.Printf("error deleting answer: %s", err.Error())
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
func (ct *controller) deleteAnswer(c *gin.Context) {
	input := c.Param("id")
	id, err := uuid.Parse(input)
	if err != nil {
		log.Printf("error parsing path question ID: %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid question id"})
		return
	}

	answer, err := ct.service.DeleteAnswer(c, id)
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

// @license.name	MIT
// @license.url	https://github.com/simple-retro/api/blob/master/LICENSE
func (c *controller) Start() {
	config := config.Get()

	// Swagger
	docs.SwaggerInfo.Title = config.Name
	docs.SwaggerInfo.Description = "API service to Simple Retro project"
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Host = fmt.Sprintf("simple-retro.ephemeral.dev.br:%d", config.Server.Port)
	docs.SwaggerInfo.BasePath = "/api"
	docs.SwaggerInfo.Schemes = []string{"http", "https"}

	router := gin.Default()

	router.Use(CORSMiddleware())

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.GET("/health", c.health)

	api := router.Group("/api")
	api.POST("/retrospective", c.createRetrospective)
	api.GET("/retrospective/:id", c.getRetrospective)
	api.PATCH("/retrospective/:id", c.updateRetrospective)
	api.DELETE("/retrospective/:id", c.deleteRetrospective)
	api.GET("/hello/:id", c.subscribeChanges)

	authorized := api.Group("/")
	authorized.Use(Authenticate())
	authorized.POST("/question", c.createQuestion)
	authorized.PATCH("/question/:id", c.updateQuestion)
	authorized.DELETE("/question/:id", c.deleteQuestion)

	authorized.POST("/answer", c.createAnswer)
	authorized.PATCH("/answer/:id", c.updateAnswer)
	authorized.DELETE("/answer/:id", c.deleteAnswer)

	router.Run(fmt.Sprintf(":%d", config.Server.Port))
}
