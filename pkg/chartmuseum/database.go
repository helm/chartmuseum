package chartmuseum

import (
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

type (
	// Database stores application data
	Database struct {
		*gorm.DB
	}

	// Org is a database model that represents an organization
	Org struct {
		gorm.Model
		Name  string `gorm:"unique_index"`
		Repos []Repo `gorm:"foreignkey:OrgID"`
	}

	// Repo is a database model that represents a repo
	Repo struct {
		gorm.Model
		Name  string `gorm:"unique_index"`
		OrgID uint
		Org   *Org `json:"-"`
	}
)

// NewDatabase creates a new Database instance
func NewDatabase() (*Database, error) {
	database, err := gorm.Open("sqlite3", "./gorm.db")
	if err != nil {
		return new(Database), err
	}
	database.AutoMigrate(&Org{}, &Repo{})
	return &Database{database}, nil
}

// adds org and repo to context (if present in URL params)
func databaseMiddleware(database *Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		if orgName := c.Param("org"); orgName != "" {
			var org Org
			if err := database.Preload("Repos").Where("name = ?", orgName).First(&org).Error; err == nil {
				c.Set("org", &org)
				if repoName := c.Param("repo"); repoName != "" {
					var repo Repo
					if err = database.Where("name = ? AND org_id = ?", repoName, org.ID).First(&repo).Error; err == nil {
						c.Set("repo", &repo)
					} else {
						c.JSON(404, gin.H{"error": "repo not found"})
						c.Abort()
						return
					}
				}
			} else {
				c.JSON(404, gin.H{"error": "org not found"})
				c.Abort()
				return
			}
		}
		c.Next()
	}
}
