package server

import (
	"api/config"
	"api/docs"
	"api/internal/repository"
	"api/internal/service"
	"api/types"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/swaggo/files"       // swagger embed files
	"github.com/swaggo/gin-swagger" // gin-swagger middleware
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
// @Summary Create Retrospective
// @Tags Retrospective
// @Accept json
// @Produce 200 {object} types.Retrospective "Retrospective Object"
// @Failure 500 {string} string "Internal error"
// @Router /retrospective [post]
func (ct *controller) createRetrospective(c *gin.Context) {
	var retrospective types.Retrospective
	if err := c.BindJSON(&retrospective); err != nil {
		log.Printf("error parsing body content: %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body content"})
		return
	}

	err := ct.service.CreateRetrospective(&retrospective)
	if err != nil {
		log.Printf("error creating pull request: %s", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, retrospective)
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

	repo, err := repository.New()
	if err != nil {
		log.Fatalf("error creating repository: %s", err.Error())
	}

	service := service.New(repo)

	controller := New(service)

	router := gin.Default()

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.GET("/health", controller.health)
	router.POST("/retrospective", controller.createRetrospective)

	router.Run(fmt.Sprintf(":%d", config.Server.Port))
}
