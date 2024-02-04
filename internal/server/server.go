package server

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"api/config"
	"api/docs"
	"api/internal/repository"
	"api/internal/service"
	"api/types"

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

// health godoc
//
//	@Summary	Show API health
//	@Tags
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

	c.SetCookie("retrospective_id", id.String(), 0, "/", ".", true, false)
	c.JSON(http.StatusOK, retro)
}

// createQuestion godoc
//
// @Summary Create Question
// @Tags Question
// @Accept json
// @Produce	json
// @Param		question	body		types.Question	true	"Create Question"
// @Success 200 {object} types.Question "Retrospective Object"
// @Failure 500 {string} string "Internal error"
// @Router /question [post]
func (ct *controller) createQuestion(c *gin.Context) {
	var question *types.Question
	if err := c.BindJSON(&question); err != nil {
		log.Printf("error parsing body content: %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body content"})
		return
	}

	err := ct.service.CreateQuestion(c, question)
	if err != nil {
		log.Printf("error creating retrospective: %s", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, question)
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

// subscribeChanges godoc
//
// @Summary Subscribe to changes via web socket
// @Tags Question
// @Accept json
// @Produce 101
// @Failure 500 {string} string "Internal error"
// @Router /hello [get]
func (ct *controller) subscribeChanges(c *gin.Context) {
	var err error
	retroIDparam := c.Param("id")
	if retroIDparam != "" {
		retroID, err := uuid.Parse(retroIDparam)
		if err != nil {
			log.Printf("error parsing retrospective_id: %s", err.Error())
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not in any retrospective"})
			return
		}
		c.Set("retrospective_id", retroID)
	}

	err = ct.service.SubscribeChanges(c, c.Writer, c.Request)
	if err != nil {
		log.Printf("error creating pull request: %s", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, "ok")
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

func Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		retroIDcookie, err := c.Cookie("retrospective_id")
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not in any retrospective"})
			return
		}

		retroID, err := uuid.Parse(retroIDcookie)
		if err != nil {
			log.Printf("error parsing retrospective_id: %s", err.Error())
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not in any retrospective"})
			return
		}

		c.Set("retrospective_id", retroID)
	}
}

// @license.name	MIT
// @license.url	https://github.com/simple-retro/api/blob/master/LICENSE
func Start() {
	config := config.Get()

	// Swagger
	docs.SwaggerInfo.Title = config.Name
	docs.SwaggerInfo.Description = "API service to Collabfy project"
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Host = "127.0.0.1:8080"
	docs.SwaggerInfo.Schemes = []string{"http", "https"}

	repo, err := repository.NewSQLite()
	if err != nil {
		log.Fatalf("error creating repository: %s", err.Error())
	}

	wsrepo, err := repository.NewWebSocket()
	if err != nil {
		log.Fatalf("error creating repository: %s", err.Error())
	}

	service := service.New(repo, wsrepo)

	controller := New(service)

	router := gin.Default()

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.GET("/health", controller.health)
	router.POST("/retrospective", controller.createRetrospective)
	router.GET("/retrospective/:id", controller.getRetrospective)
	router.PATCH("/retrospective/:id", controller.updateRetrospective)
	router.DELETE("/retrospective/:id", controller.deleteRetrospective)
	router.GET("/hello", controller.subscribeChanges)
	router.GET("/hello/:id", controller.subscribeChanges)

	authorized := router.Group("/")
	authorized.Use(Authenticate())
	authorized.POST("/question", controller.createQuestion)

	router.Run(fmt.Sprintf(":%d", config.Server.Port))
}
