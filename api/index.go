package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/go-sql-driver/mysql"
)

type Recipe struct {
	ID               int      `json:"id"`
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	Image            string   `json:"image"`
	PrepTimeMinutes  *int     `json:"prep_time_minutes"`
	CookTimeMinutes  *int     `json:"cook_time_minutes"`
	TotalTimeMinutes *int     `json:"total_time_minutes"`
	Servings         *int     `json:"servings"`
	Rating           *float64 `json:"rating"`
	Ingredients      []string `json:"ingredients"`
	Instructions     []string `json:"instructions"`
	Calories         *int     `json:"calories"`
	Protein          *float64 `json:"protein"`
	Fat              *float64 `json:"fat"`
	Carbs            *float64 `json:"carbs"`
	Fiber            *float64 `json:"fiber"`
	Sodium           *float64 `json:"sodium"`
}

type DietPlan struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Filters     map[string]interface{} `json:"filters"`
}

type MCPRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

type MCPResource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MimeType    string `json:"mimeType"`
}

var db *sql.DB
var dietPlans = map[string]DietPlan{
	"keto": {
		Name:        "Ketogenic Diet",
		Description: "High fat, very low carb diet for ketosis",
		Filters: map[string]interface{}{
			"max_carbs":  20,
			"min_fat":    15,
			"sort_by":    "fat",
			"sort_order": "desc",
		},
	},
	"paleo": {
		Name:        "Paleo Diet",
		Description: "Whole foods, no processed ingredients",
		Filters: map[string]interface{}{
			"exclude_ingredients": []string{"wheat", "grain", "dairy", "sugar", "legume", "bean"},
			"sort_by":             "protein",
			"sort_order":          "desc",
		},
	},
	"mediterranean": {
		Name:        "Mediterranean Diet",
		Description: "Heart-healthy with olive oil, fish, and vegetables",
		Filters: map[string]interface{}{
			"include_ingredients": []string{"olive", "fish", "vegetable", "fruit", "nut"},
			"max_sodium":          1500,
			"sort_by":             "rating",
			"sort_order":          "desc",
		},
	},
	"vegan": {
		Name:        "Vegan Diet",
		Description: "Plant-based, no animal products",
		Filters: map[string]interface{}{
			"exclude_ingredients": []string{"meat", "chicken", "beef", "pork", "fish", "dairy", "milk", "cheese", "egg", "butter"},
			"sort_by":             "fiber",
			"sort_order":          "desc",
		},
	},
	"vegetarian": {
		Name:        "Vegetarian Diet",
		Description: "No meat, but includes dairy and eggs",
		Filters: map[string]interface{}{
			"exclude_ingredients": []string{"meat", "chicken", "beef", "pork", "fish", "seafood"},
			"sort_by":             "protein",
			"sort_order":          "desc",
		},
	},
	"low_carb": {
		Name:        "Low Carb Diet",
		Description: "Reduced carbohydrate intake",
		Filters: map[string]interface{}{
			"max_carbs":  50,
			"sort_by":    "carbs",
			"sort_order": "asc",
		},
	},
	"high_protein": {
		Name:        "High Protein Diet",
		Description: "Protein-rich foods for muscle building",
		Filters: map[string]interface{}{
			"min_protein": 20,
			"sort_by":     "protein",
			"sort_order":  "desc",
		},
	},
}

func initDB() {
	godotenv.Load()
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
		os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_NAME"))

	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
}

func handleMCP(c *gin.Context) {
	var req MCPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, MCPResponse{JSONRPC: "2.0", ID: req.ID, Error: &MCPError{-32700, "Parse error"}})
		return
	}

	switch req.Method {
	case "initialize":
		c.JSON(200, MCPResponse{
			JSONRPC: "2.0", ID: req.ID,
			Result: map[string]interface{}{
				"protocolVersion": "2025-07-05",
				"capabilities": map[string]interface{}{
					"tools":     map[string]interface{}{"listChanged": false},
					"resources": map[string]interface{}{"subscribe": false, "listChanged": false},
				},
				"serverInfo": map[string]interface{}{"name": "recipe-server", "version": "1.0.0"},
			},
		})
	case "tools/list":
		tools := []MCPTool{
			{
				Name:        "search_recipes",
				Description: "Search recipes by criteria",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"search":             map[string]interface{}{"type": "string"},
						"diet":               map[string]interface{}{"type": "string"},
						"include_ingredients": map[string]interface{}{"type": "string"},
						"exclude_ingredients": map[string]interface{}{"type": "string"},
						"max_calories":       map[string]interface{}{"type": "integer"},
						"min_protein":        map[string]interface{}{"type": "number"},
						"max_carbs":          map[string]interface{}{"type": "number"},
						"sort_by":            map[string]interface{}{"type": "string"},
						"sort_order":         map[string]interface{}{"type": "string"},
					},
				},
			},
			{
				Name:        "get_recipe",
				Description: "Get recipe by ID",
				InputSchema: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{"id": map[string]interface{}{"type": "integer"}},
					"required":   []string{"id"},
				},
			},
		}
		c.JSON(200, MCPResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]interface{}{"tools": tools}})
	case "tools/call":
		params := req.Params.(map[string]interface{})
		name := params["name"].(string)
		args := params["arguments"].(map[string]interface{})

		var result string
		switch name {
		case "search_recipes":
			result = searchRecipesMCP(args)
		case "get_recipe":
			if id, ok := args["id"].(float64); ok {
				result = getRecipeMCP(int(id))
			} else {
				c.JSON(200, MCPResponse{JSONRPC: "2.0", ID: req.ID, Error: &MCPError{-32602, "Invalid recipe ID"}})
				return
			}
		default:
			c.JSON(200, MCPResponse{JSONRPC: "2.0", ID: req.ID, Error: &MCPError{-32601, "Tool not found"}})
			return
		}

		c.JSON(200, MCPResponse{
			JSONRPC: "2.0", ID: req.ID,
			Result: map[string]interface{}{"content": []map[string]interface{}{{"type": "text", "text": result}}},
		})
	default:
		c.JSON(200, MCPResponse{JSONRPC: "2.0", ID: req.ID, Error: &MCPError{-32601, "Method not found"}})
	}
}

func searchRecipesMCP(args map[string]interface{}) string {
	query := "SELECT id, name, description, image, prep_time_minutes, cook_time_minutes, total_time_minutes, servings, rating, ingredients, instructions, calories, protein, fat, carbs, fiber, sodium FROM recipes WHERE 1=1"
	sqlArgs := []interface{}{}

	if diet, ok := args["diet"].(string); ok && diet != "" {
		if plan, exists := dietPlans[diet]; exists {
			query, sqlArgs = applyDietFilters(query, sqlArgs, plan.Filters)
		}
	}

	if search, ok := args["search"].(string); ok && search != "" {
		query += " AND (name LIKE ? OR description LIKE ?)"
		term := "%" + search + "%"
		sqlArgs = append(sqlArgs, term, term)
	}

	if inc, ok := args["include_ingredients"].(string); ok && inc != "" {
		for _, ing := range strings.Split(inc, ",") {
			query += " AND ingredients LIKE ?"
			sqlArgs = append(sqlArgs, "%"+strings.TrimSpace(ing)+"%")
		}
	}

	if exc, ok := args["exclude_ingredients"].(string); ok && exc != "" {
		for _, ing := range strings.Split(exc, ",") {
			query += " AND ingredients NOT LIKE ?"
			sqlArgs = append(sqlArgs, "%"+strings.TrimSpace(ing)+"%")
		}
	}

	if val, ok := args["max_calories"].(float64); ok {
		query += " AND calories <= ?"
		sqlArgs = append(sqlArgs, val)
	}

	if val, ok := args["min_protein"].(float64); ok {
		query += " AND protein >= ?"
		sqlArgs = append(sqlArgs, val)
	}

	if val, ok := args["max_carbs"].(float64); ok {
		query += " AND carbs <= ?"
		sqlArgs = append(sqlArgs, val)
	}

	sortBy, sortOrder := "id", "asc"
	if val, ok := args["sort_by"].(string); ok && val != "" {
		sortBy = val
	}
	if val, ok := args["sort_order"].(string); ok && val != "" {
		sortOrder = val
	}

	validCols := map[string]bool{"id": true, "name": true, "rating": true, "calories": true, "protein": true, "carbs": true}
	if validCols[sortBy] {
		if sortOrder == "desc" {
			query += " ORDER BY " + sortBy + " DESC"
		} else {
			query += " ORDER BY " + sortBy + " ASC"
		}
	}

	query += " LIMIT 20"

	rows, err := db.Query(query, sqlArgs...)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	defer rows.Close()

	var recipes []Recipe
	for rows.Next() {
		var r Recipe
		var ingredientsJSON, instructionsJSON string
		rows.Scan(&r.ID, &r.Name, &r.Description, &r.Image, &r.PrepTimeMinutes, &r.CookTimeMinutes,
			&r.TotalTimeMinutes, &r.Servings, &r.Rating, &ingredientsJSON, &instructionsJSON,
			&r.Calories, &r.Protein, &r.Fat, &r.Carbs, &r.Fiber, &r.Sodium)

		if ingredientsJSON != "" {
			json.Unmarshal([]byte(ingredientsJSON), &r.Ingredients)
		}
		if instructionsJSON != "" {
			json.Unmarshal([]byte(instructionsJSON), &r.Instructions)
		}
		recipes = append(recipes, r)
	}

	result, _ := json.MarshalIndent(map[string]interface{}{"recipes": recipes, "count": len(recipes)}, "", "  ")
	return string(result)
}

func getRecipeMCP(id int) string {
	var r Recipe
	var ingredientsJSON, instructionsJSON string

	err := db.QueryRow("SELECT id, name, description, image, prep_time_minutes, cook_time_minutes, total_time_minutes, servings, rating, ingredients, instructions, calories, protein, fat, carbs, fiber, sodium FROM recipes WHERE id = ?", id).Scan(
		&r.ID, &r.Name, &r.Description, &r.Image, &r.PrepTimeMinutes, &r.CookTimeMinutes,
		&r.TotalTimeMinutes, &r.Servings, &r.Rating, &ingredientsJSON, &instructionsJSON,
		&r.Calories, &r.Protein, &r.Fat, &r.Carbs, &r.Fiber, &r.Sodium)

	if err == sql.ErrNoRows {
		return fmt.Sprintf("Recipe %d not found", id)
	}
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	if ingredientsJSON != "" {
		json.Unmarshal([]byte(ingredientsJSON), &r.Ingredients)
	}
	if instructionsJSON != "" {
		json.Unmarshal([]byte(instructionsJSON), &r.Instructions)
	}

	result, _ := json.MarshalIndent(r, "", "  ")
	return string(result)
}

func applyDietFilters(query string, args []interface{}, filters map[string]interface{}) (string, []interface{}) {
	for key, value := range filters {
		switch key {
		case "max_carbs", "min_carbs", "max_calories", "min_calories", "max_protein", "min_protein", "max_fat", "min_fat", "max_sodium", "min_sodium":
			if val, ok := value.(int); ok {
				op := ">="
				if strings.HasPrefix(key, "max_") {
					op = "<="
				}
				query += fmt.Sprintf(" AND %s %s ?", strings.TrimPrefix(strings.TrimPrefix(key, "max_"), "min_"), op)
				args = append(args, val)
			}
		case "exclude_ingredients", "include_ingredients":
			if ingredients, ok := value.([]string); ok {
				op := "LIKE"
				if key == "exclude_ingredients" {
					op = "NOT LIKE"
				}
				for _, ingredient := range ingredients {
					query += fmt.Sprintf(" AND ingredients %s ?", op)
					args = append(args, "%"+ingredient+"%")
				}
			}
		}
	}
	return query, args
}

func Handler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		initDB()
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	router.POST("/mcp", handleMCP)
	router.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })

	router.ServeHTTP(w, r)
}

func main() {
	initDB()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.POST("/mcp", handleMCP)
	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	r.Run(":" + port)
}