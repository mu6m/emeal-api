package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/go-sql-driver/mysql"
)

type Recipe struct {
	ID               int               `json:"id"`
	Name             string            `json:"name"`
	Description      string            `json:"description"`
	Image            string            `json:"image"`
	PrepTimeMinutes  *int              `json:"prep_time_minutes"`
	CookTimeMinutes  *int              `json:"cook_time_minutes"`
	TotalTimeMinutes *int              `json:"total_time_minutes"`
	Servings         *int              `json:"servings"`
	Rating           *float64          `json:"rating"`
	Ingredients      []string          `json:"ingredients"`
	Instructions     []string          `json:"instructions"`
	Calories         *int              `json:"calories"`
	Protein          *float64          `json:"protein"`
	Fat              *float64          `json:"fat"`
	Carbs            *float64          `json:"carbs"`
	Fiber            *float64          `json:"fiber"`
	Sodium           *float64          `json:"sodium"`
}

type DietPlan struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Filters     map[string]interface{} `json:"filters"`
}

// MCP Protocol Types
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

type MCPToolCall struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
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
			"max_carbs": 20,
			"min_fat": 15,
			"sort_by": "fat",
			"sort_order": "desc",
		},
	},
	"paleo": {
		Name:        "Paleo Diet",
		Description: "Whole foods, no processed ingredients",
		Filters: map[string]interface{}{
			"exclude_ingredients": []string{"wheat", "grain", "dairy", "sugar", "legume", "bean"},
			"sort_by": "protein",
			"sort_order": "desc",
		},
	},
	"mediterranean": {
		Name:        "Mediterranean Diet",
		Description: "Heart-healthy with olive oil, fish, and vegetables",
		Filters: map[string]interface{}{
			"include_ingredients": []string{"olive", "fish", "vegetable", "fruit", "nut"},
			"max_sodium": 1500,
			"sort_by": "rating",
			"sort_order": "desc",
		},
	},
	"vegan": {
		Name:        "Vegan Diet",
		Description: "Plant-based, no animal products",
		Filters: map[string]interface{}{
			"exclude_ingredients": []string{"meat", "chicken", "beef", "pork", "fish", "dairy", "milk", "cheese", "egg", "butter"},
			"sort_by": "fiber",
			"sort_order": "desc",
		},
	},
	"vegetarian": {
		Name:        "Vegetarian Diet",
		Description: "No meat, but includes dairy and eggs",
		Filters: map[string]interface{}{
			"exclude_ingredients": []string{"meat", "chicken", "beef", "pork", "fish", "seafood"},
			"sort_by": "protein",
			"sort_order": "desc",
		},
	},
	"low_carb": {
		Name:        "Low Carb Diet",
		Description: "Reduced carbohydrate intake",
		Filters: map[string]interface{}{
			"max_carbs": 50,
			"sort_by": "carbs",
			"sort_order": "asc",
		},
	},
	"high_protein": {
		Name:        "High Protein Diet",
		Description: "Protein-rich foods for muscle building",
		Filters: map[string]interface{}{
			"min_protein": 20,
			"sort_by": "protein",
			"sort_order": "desc",
		},
	},
	"low_sodium": {
		Name:        "Low Sodium Diet",
		Description: "Heart-healthy, reduced sodium intake",
		Filters: map[string]interface{}{
			"max_sodium": 1000,
			"sort_by": "sodium",
			"sort_order": "asc",
		},
	},
	"diabetic": {
		Name:        "Diabetic Friendly",
		Description: "Low sugar, controlled carbs",
		Filters: map[string]interface{}{
			"max_carbs": 45,
			"exclude_ingredients": []string{"sugar", "honey", "syrup", "candy"},
			"sort_by": "carbs",
			"sort_order": "asc",
		},
	},
	"heart_healthy": {
		Name:        "Heart Healthy",
		Description: "Low sodium, healthy fats",
		Filters: map[string]interface{}{
			"max_sodium": 1200,
			"min_fiber": 5,
			"exclude_ingredients": []string{"fried", "processed"},
			"sort_by": "fiber",
			"sort_order": "desc",
		},
	},
}

func initDB() {
	godotenv.Load()
	
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	database := os.Getenv("DB_NAME")
	
	dsn := user + ":" + password + "@tcp(" + host + ":" + port + ")/" + database
	
	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
}

// MCP Server Handlers
func handleMCPRequest(c *gin.Context) {
	var req MCPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32700,
				Message: "Parse error",
			},
		})
		return
	}

	switch req.Method {
	case "initialize":
		handleMCPInitialize(c, req)
	case "tools/list":
		handleMCPToolsList(c, req)
	case "tools/call":
		handleMCPToolCall(c, req)
	case "resources/list":
		handleMCPResourcesList(c, req)
	case "resources/read":
		handleMCPResourcesRead(c, req)
	default:
		c.JSON(http.StatusOK, MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32601,
				Message: "Method not found",
			},
		})
	}
}

func handleMCPInitialize(c *gin.Context, req MCPRequest) {
	result := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{
				"listChanged": false,
			},
			"resources": map[string]interface{}{
				"subscribe": false,
				"listChanged": false,
			},
		},
		"serverInfo": map[string]interface{}{
			"name":    "recipe-server",
			"version": "1.0.0",
		},
	}

	c.JSON(http.StatusOK, MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	})
}

func handleMCPToolsList(c *gin.Context, req MCPRequest) {
	tools := []MCPTool{
		{
			Name:        "search_recipes",
			Description: "Search for recipes based on various criteria including diet plans, ingredients, nutritional values, and preparation time",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"search": map[string]interface{}{
						"type":        "string",
						"description": "Text search in recipe name or description",
					},
					"diet": map[string]interface{}{
						"type":        "string",
						"description": "Diet plan filter (keto, paleo, mediterranean, vegan, vegetarian, low_carb, high_protein, low_sodium, diabetic, heart_healthy)",
					},
					"include_ingredients": map[string]interface{}{
						"type":        "string",
						"description": "Comma-separated ingredients to include",
					},
					"exclude_ingredients": map[string]interface{}{
						"type":        "string",
						"description": "Comma-separated ingredients to exclude",
					},
					"max_calories": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum calories per serving",
					},
					"min_protein": map[string]interface{}{
						"type":        "number",
						"description": "Minimum protein in grams",
					},
					"max_carbs": map[string]interface{}{
						"type":        "number",
						"description": "Maximum carbs in grams",
					},
					"max_prep_time": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum preparation time in minutes",
					},
					"sort_by": map[string]interface{}{
						"type":        "string",
						"description": "Sort field (rating, calories, protein, carbs, prep_time_minutes, etc.)",
					},
					"sort_order": map[string]interface{}{
						"type":        "string",
						"description": "Sort order (asc or desc)",
					},
				},
				"additionalProperties": true,
			},
		},
		{
			Name:        "get_recipe",
			Description: "Get detailed information about a specific recipe by ID",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "integer",
						"description": "Recipe ID",
					},
				},
				"required": []string{"id"},
			},
		},
		{
			Name:        "get_diet_plans",
			Description: "Get list of available diet plans with their descriptions and filters",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{},
			},
		},
	}

	c.JSON(http.StatusOK, MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"tools": tools,
		},
	})
}

func handleMCPToolCall(c *gin.Context, req MCPRequest) {
	params, ok := req.Params.(map[string]interface{})
	if !ok {
		c.JSON(http.StatusOK, MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32602,
				Message: "Invalid params",
			},
		})
		return
	}

	name, _ := params["name"].(string)
	arguments, _ := params["arguments"].(map[string]interface{})

	switch name {
	case "search_recipes":
		result := mcpSearchRecipes(arguments)
		c.JSON(http.StatusOK, MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": result,
					},
				},
			},
		})
	case "get_recipe":
		if id, ok := arguments["id"].(float64); ok {
			result := mcpGetRecipe(int(id))
			c.JSON(http.StatusOK, MCPResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: map[string]interface{}{
					"content": []map[string]interface{}{
						{
							"type": "text",
							"text": result,
						},
					},
				},
			})
		} else {
			c.JSON(http.StatusOK, MCPResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &MCPError{
					Code:    -32602,
					Message: "Invalid recipe ID",
				},
			})
		}
	case "get_diet_plans":
		result := mcpGetDietPlans()
		c.JSON(http.StatusOK, MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": result,
					},
				},
			},
		})
	default:
		c.JSON(http.StatusOK, MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32601,
				Message: "Tool not found",
			},
		})
	}
}

func handleMCPResourcesList(c *gin.Context, req MCPRequest) {
	resources := []MCPResource{
		{
			URI:         "recipe://diet-plans",
			Name:        "Diet Plans",
			Description: "Available diet plans and their configurations",
			MimeType:    "application/json",
		},
	}

	c.JSON(http.StatusOK, MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"resources": resources,
		},
	})
}

func handleMCPResourcesRead(c *gin.Context, req MCPRequest) {
	params, ok := req.Params.(map[string]interface{})
	if !ok {
		c.JSON(http.StatusOK, MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32602,
				Message: "Invalid params",
			},
		})
		return
	}

	uri, _ := params["uri"].(string)

	switch uri {
	case "recipe://diet-plans":
		data, _ := json.MarshalIndent(dietPlans, "", "  ")
		c.JSON(http.StatusOK, MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"contents": []map[string]interface{}{
					{
						"uri":      uri,
						"mimeType": "application/json",
						"text":     string(data),
					},
				},
			},
		})
	default:
		c.JSON(http.StatusOK, MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32601,
				Message: "Resource not found",
			},
		})
	}
}

// MCP Tool Implementation Functions
func mcpSearchRecipes(args map[string]interface{}) string {
	query := "SELECT id, name, description, image, prep_time_minutes, cook_time_minutes, total_time_minutes, servings, rating, ingredients, instructions, calories, protein, fat, carbs, fiber, sodium FROM recipes WHERE 1=1"
	sqlArgs := []interface{}{}

	// Apply diet plan filters if specified
	if diet, ok := args["diet"].(string); ok && diet != "" {
		if plan, exists := dietPlans[diet]; exists {
			query, sqlArgs = applyDietFilters(query, sqlArgs, plan.Filters)
		}
	}

	// Apply other filters
	filters := map[string]string{
		"search": "AND (name LIKE ? OR description LIKE ?)",
		"include_ingredients": "AND ingredients LIKE ?",
		"exclude_ingredients": "AND ingredients NOT LIKE ?",
		"min_calories": "AND calories >= ?",
		"max_calories": "AND calories <= ?",
		"min_protein": "AND protein >= ?",
		"max_protein": "AND protein <= ?",
		"min_carbs": "AND carbs >= ?",
		"max_carbs": "AND carbs <= ?",
		"max_prep_time": "AND prep_time_minutes <= ?",
	}

	for key, condition := range filters {
		if value, ok := args[key]; ok && value != nil {
			switch key {
			case "search":
				if str, ok := value.(string); ok && str != "" {
					query += " " + condition
					searchTerm := "%" + str + "%"
					sqlArgs = append(sqlArgs, searchTerm, searchTerm)
				}
			case "include_ingredients", "exclude_ingredients":
				if str, ok := value.(string); ok && str != "" {
					ingredients := strings.Split(str, ",")
					for _, ingredient := range ingredients {
						query += " " + condition
						sqlArgs = append(sqlArgs, "%"+strings.TrimSpace(ingredient)+"%")
					}
				}
			default:
				if str, ok := value.(string); ok && str != "" {
					query += " " + condition
					if val, err := strconv.ParseFloat(str, 64); err == nil {
						sqlArgs = append(sqlArgs, val)
					}
				} else if num, ok := value.(float64); ok {
					query += " " + condition
					sqlArgs = append(sqlArgs, num)
				}
			}
		}
	}

	// Sorting
	sortBy := "id"
	sortOrder := "asc"
	if val, ok := args["sort_by"].(string); ok && val != "" {
		sortBy = val
	}
	if val, ok := args["sort_order"].(string); ok && val != "" {
		sortOrder = val
	}

	validSortColumns := map[string]bool{
		"id": true, "name": true, "prep_time_minutes": true, "cook_time_minutes": true,
		"total_time_minutes": true, "servings": true, "rating": true, "calories": true,
		"protein": true, "fat": true, "carbs": true, "fiber": true, "sodium": true,
	}

	if validSortColumns[sortBy] {
		if sortOrder == "desc" {
			query += " ORDER BY " + sortBy + " DESC"
		} else {
			query += " ORDER BY " + sortBy + " ASC"
		}
	}

	query += " LIMIT 20"

	rows, err := db.Query(query, sqlArgs...)
	if err != nil {
		return fmt.Sprintf("Error searching recipes: %v", err)
	}
	defer rows.Close()

	var recipes []Recipe
	for rows.Next() {
		var recipe Recipe
		var ingredientsJSON, instructionsJSON string

		err := rows.Scan(&recipe.ID, &recipe.Name, &recipe.Description, &recipe.Image,
			&recipe.PrepTimeMinutes, &recipe.CookTimeMinutes, &recipe.TotalTimeMinutes,
			&recipe.Servings, &recipe.Rating, &ingredientsJSON, &instructionsJSON,
			&recipe.Calories, &recipe.Protein, &recipe.Fat, &recipe.Carbs, &recipe.Fiber, &recipe.Sodium)

		if err != nil {
			continue
		}

		// Parse JSON strings into slices
		if ingredientsJSON != "" {
			json.Unmarshal([]byte(ingredientsJSON), &recipe.Ingredients)
		}
		if instructionsJSON != "" {
			json.Unmarshal([]byte(instructionsJSON), &recipe.Instructions)
		}

		recipes = append(recipes, recipe)
	}

	result, _ := json.MarshalIndent(map[string]interface{}{
		"recipes": recipes,
		"count":   len(recipes),
	}, "", "  ")

	return string(result)
}

func mcpGetRecipe(id int) string {
	query := "SELECT id, name, description, image, prep_time_minutes, cook_time_minutes, total_time_minutes, servings, rating, ingredients, instructions, calories, protein, fat, carbs, fiber, sodium FROM recipes WHERE id = ?"

	var recipe Recipe
	var ingredientsJSON, instructionsJSON string

	err := db.QueryRow(query, id).Scan(
		&recipe.ID, &recipe.Name, &recipe.Description, &recipe.Image,
		&recipe.PrepTimeMinutes, &recipe.CookTimeMinutes, &recipe.TotalTimeMinutes,
		&recipe.Servings, &recipe.Rating, &ingredientsJSON, &instructionsJSON,
		&recipe.Calories, &recipe.Protein, &recipe.Fat, &recipe.Carbs, &recipe.Fiber, &recipe.Sodium)

	if err == sql.ErrNoRows {
		return fmt.Sprintf("Recipe with ID %d not found", id)
	}

	if err != nil {
		return fmt.Sprintf("Error fetching recipe: %v", err)
	}

	// Parse JSON strings into slices
	if ingredientsJSON != "" {
		json.Unmarshal([]byte(ingredientsJSON), &recipe.Ingredients)
	}
	if instructionsJSON != "" {
		json.Unmarshal([]byte(instructionsJSON), &recipe.Instructions)
	}

	result, _ := json.MarshalIndent(recipe, "", "  ")
	return string(result)
}

func mcpGetDietPlans() string {
	result, _ := json.MarshalIndent(map[string]interface{}{
		"diet_plans": dietPlans,
	}, "", "  ")
	return string(result)
}

// Original API Handlers (unchanged)
func searchRecipes(c *gin.Context) {
	query := "SELECT id, name, description, image, prep_time_minutes, cook_time_minutes, total_time_minutes, servings, rating, ingredients, instructions, calories, protein, fat, carbs, fiber, sodium FROM recipes WHERE 1=1"
	args := []interface{}{}
	
	// Apply diet plan filters if specified
	if diet := c.Query("diet"); diet != "" {
		if plan, exists := dietPlans[diet]; exists {
			query, args = applyDietFilters(query, args, plan.Filters)
		}
	}
	
	// Text search
	if search := c.Query("search"); search != "" {
		query += " AND (name LIKE ? OR description LIKE ?)"
		searchTerm := "%" + search + "%"
		args = append(args, searchTerm, searchTerm)
	}
	
	// Ingredient filters
	if includeIngredients := c.Query("include_ingredients"); includeIngredients != "" {
		ingredients := strings.Split(includeIngredients, ",")
		for _, ingredient := range ingredients {
			query += " AND ingredients LIKE ?"
			args = append(args, "%"+strings.TrimSpace(ingredient)+"%")
		}
	}
	
	if excludeIngredients := c.Query("exclude_ingredients"); excludeIngredients != "" {
		ingredients := strings.Split(excludeIngredients, ",")
		for _, ingredient := range ingredients {
			query += " AND ingredients NOT LIKE ?"
			args = append(args, "%"+strings.TrimSpace(ingredient)+"%")
		}
	}
	
	// Numeric filters
	if minCalories := c.Query("min_calories"); minCalories != "" {
		if val, err := strconv.Atoi(minCalories); err == nil {
			query += " AND calories >= ?"
			args = append(args, val)
		}
	}
	
	if maxCalories := c.Query("max_calories"); maxCalories != "" {
		if val, err := strconv.Atoi(maxCalories); err == nil {
			query += " AND calories <= ?"
			args = append(args, val)
		}
	}
	
	if minProtein := c.Query("min_protein"); minProtein != "" {
		if val, err := strconv.ParseFloat(minProtein, 64); err == nil {
			query += " AND protein >= ?"
			args = append(args, val)
		}
	}
	
	if maxProtein := c.Query("max_protein"); maxProtein != "" {
		if val, err := strconv.ParseFloat(maxProtein, 64); err == nil {
			query += " AND protein <= ?"
			args = append(args, val)
		}
	}
	
	if minFat := c.Query("min_fat"); minFat != "" {
		if val, err := strconv.ParseFloat(minFat, 64); err == nil {
			query += " AND fat >= ?"
			args = append(args, val)
		}
	}
	
	if maxFat := c.Query("max_fat"); maxFat != "" {
		if val, err := strconv.ParseFloat(maxFat, 64); err == nil {
			query += " AND fat <= ?"
			args = append(args, val)
		}
	}
	
	if minCarbs := c.Query("min_carbs"); minCarbs != "" {
		if val, err := strconv.ParseFloat(minCarbs, 64); err == nil {
			query += " AND carbs >= ?"
			args = append(args, val)
		}
	}
	
	if maxCarbs := c.Query("max_carbs"); maxCarbs != "" {
		if val, err := strconv.ParseFloat(maxCarbs, 64); err == nil {
			query += " AND carbs <= ?"
			args = append(args, val)
		}
	}
	
	if minFiber := c.Query("min_fiber"); minFiber != "" {
		if val, err := strconv.ParseFloat(minFiber, 64); err == nil {
			query += " AND fiber >= ?"
			args = append(args, val)
		}
	}
	
	if maxFiber := c.Query("max_fiber"); maxFiber != "" {
		if val, err := strconv.ParseFloat(maxFiber, 64); err == nil {
			query += " AND fiber <= ?"
			args = append(args, val)
		}
	}
	
	if minSodium := c.Query("min_sodium"); minSodium != "" {
		if val, err := strconv.ParseFloat(minSodium, 64); err == nil {
			query += " AND sodium >= ?"
			args = append(args, val)
		}
	}
	
	if maxSodium := c.Query("max_sodium"); maxSodium != "" {
		if val, err := strconv.ParseFloat(maxSodium, 64); err == nil {
			query += " AND sodium <= ?"
			args = append(args, val)
		}
	}
	
	if minPrepTime := c.Query("min_prep_time"); minPrepTime != "" {
		if val, err := strconv.Atoi(minPrepTime); err == nil {
			query += " AND prep_time_minutes >= ?"
			args = append(args, val)
		}
	}
	
	if maxPrepTime := c.Query("max_prep_time"); maxPrepTime != "" {
		if val, err := strconv.Atoi(maxPrepTime); err == nil {
			query += " AND prep_time_minutes <= ?"
			args = append(args, val)
		}
	}
	
	if minCookTime := c.Query("min_cook_time"); minCookTime != "" {
		if val, err := strconv.Atoi(minCookTime); err == nil {
			query += " AND cook_time_minutes >= ?"
			args = append(args, val)
		}
	}
	
	if maxCookTime := c.Query("max_cook_time"); maxCookTime != "" {
		if val, err := strconv.Atoi(maxCookTime); err == nil {
			query += " AND cook_time_minutes <= ?"
			args = append(args, val)
		}
	}
	
	if minTotalTime := c.Query("min_total_time"); minTotalTime != "" {
		if val, err := strconv.Atoi(minTotalTime); err == nil {
			query += " AND total_time_minutes >= ?"
			args = append(args, val)
		}
	}
	
	if maxTotalTime := c.Query("max_total_time"); maxTotalTime != "" {
		if val, err := strconv.Atoi(maxTotalTime); err == nil {
			query += " AND total_time_minutes <= ?"
			args = append(args, val)
		}
	}
	
	if minServings := c.Query("min_servings"); minServings != "" {
		if val, err := strconv.Atoi(minServings); err == nil {
			query += " AND servings >= ?"
			args = append(args, val)
		}
	}
	
	if maxServings := c.Query("max_servings"); maxServings != "" {
		if val, err := strconv.Atoi(maxServings); err == nil {
			query += " AND servings <= ?"
			args = append(args, val)
		}
	}
	
	if minRating := c.Query("min_rating"); minRating != "" {
		if val, err := strconv.ParseFloat(minRating, 64); err == nil {
			query += " AND rating >= ?"
			args = append(args, val)
		}
	}
	
	if maxRating := c.Query("max_rating"); maxRating != "" {
		if val, err := strconv.ParseFloat(maxRating, 64); err == nil {
			query += " AND rating <= ?"
			args = append(args, val)
		}
	}
	
	// Sorting
	sortBy := c.DefaultQuery("sort_by", "id")
	sortOrder := c.DefaultQuery("sort_order", "asc")
	
	validSortColumns := map[string]bool{
		"id": true, "name": true, "prep_time_minutes": true, "cook_time_minutes": true,
		"total_time_minutes": true, "servings": true, "rating": true, "calories": true,
		"protein": true, "fat": true, "carbs": true, "fiber": true, "sodium": true,
	}
	
	if validSortColumns[sortBy] {
		if sortOrder == "desc" {
			query += " ORDER BY " + sortBy + " DESC"
		} else {
			query += " ORDER BY " + sortBy + " ASC"
		}
	}
	
	query += " LIMIT 100"
	
	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	
	var recipes []Recipe
	for rows.Next() {
		var recipe Recipe
		var ingredientsJSON, instructionsJSON string
		
		err := rows.Scan(&recipe.ID, &recipe.Name, &recipe.Description, &recipe.Image,
			&recipe.PrepTimeMinutes, &recipe.CookTimeMinutes, &recipe.TotalTimeMinutes,
			&recipe.Servings, &recipe.Rating, &ingredientsJSON, &instructionsJSON,
			&recipe.Calories, &recipe.Protein, &recipe.Fat, &recipe.Carbs, &recipe.Fiber, &recipe.Sodium)
		
		if err != nil {
			continue
		}
		
		// Parse JSON strings into slices
		if ingredientsJSON != "" {
			json.Unmarshal([]byte(ingredientsJSON), &recipe.Ingredients)
		}
		if instructionsJSON != "" {
			json.Unmarshal([]byte(instructionsJSON), &recipe.Instructions)
		}
		
		recipes = append(recipes, recipe)
	}
	
	response := gin.H{
		"recipes": recipes,
		"count":   len(recipes),
	}
	
	// Include diet plan info if used
	if diet := c.Query("diet"); diet != "" {
		if plan, exists := dietPlans[diet]; exists {
			response["diet_plan"] = plan
		}
	}
	
	c.JSON(http.StatusOK, response)
}

func applyDietFilters(query string, args []interface{}, filters map[string]interface{}) (string, []interface{}) {
	for key, value := range filters {
		switch key {
		case "max_carbs":
			if val, ok := value.(int); ok {
				query += " AND carbs <= ?"
				args = append(args, val)
			}
		case "min_carbs":
			if val, ok := value.(int); ok {
				query += " AND carbs >= ?"
				args = append(args, val)
			}
		case "max_calories":
			if val, ok := value.(int); ok {
				query += " AND calories <= ?"
				args = append(args, val)
			}
		case "min_calories":
			if val, ok := value.(int); ok {
				query += " AND calories >= ?"
				args = append(args, val)
			}
		case "max_protein":
			if val, ok := value.(int); ok {
				query += " AND protein <= ?"
				args = append(args, val)
			}
		case "min_protein":
			if val, ok := value.(int); ok {
				query += " AND protein >= ?"
				args = append(args, val)
			}
		case "max_fat":
			if val, ok := value.(int); ok {
				query += " AND fat <= ?"
				args = append(args, val)
			}
		case "min_fat":
			if val, ok := value.(int); ok {
				query += " AND fat >= ?"
				args = append(args, val)
			}
		case "max_fiber":
			if val, ok := value.(int); ok {
				query += " AND fiber <= ?"
				args = append(args, val)
			}
		case "min_fiber":
			if val, ok := value.(int); ok {
				query += " AND fiber >= ?"
				args = append(args, val)
			}
		case "max_sodium":
			if val, ok := value.(int); ok {
				query += " AND sodium <= ?"
				args = append(args, val)
			}
		case "min_sodium":
			if val, ok := value.(int); ok {
				query += " AND sodium >= ?"
				args = append(args, val)
			}
		case "exclude_ingredients":
			if ingredients, ok := value.([]string); ok {
				for _, ingredient := range ingredients {
					query += " AND ingredients NOT LIKE ?"
					args = append(args, "%"+ingredient+"%")
				}
			}
		case "include_ingredients":
			if ingredients, ok := value.([]string); ok {
				for _, ingredient := range ingredients {
					query += " AND ingredients LIKE ?"
					args = append(args, "%"+ingredient+"%")
				}
			}
		}
	}
	return query, args
}

func getDietPlans(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"diet_plans": dietPlans})
}

func getRecipeByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid recipe ID"})
		return
	}
	
	query := "SELECT id, name, description, image, prep_time_minutes, cook_time_minutes, total_time_minutes, servings, rating, ingredients, instructions, calories, protein, fat, carbs, fiber, sodium FROM recipes WHERE id = ?"
	
	var recipe Recipe
	var ingredientsJSON, instructionsJSON string
	
	err = db.QueryRow(query, id).Scan(
		&recipe.ID, &recipe.Name, &recipe.Description, &recipe.Image,
		&recipe.PrepTimeMinutes, &recipe.CookTimeMinutes, &recipe.TotalTimeMinutes,
		&recipe.Servings, &recipe.Rating, &ingredientsJSON, &instructionsJSON,
		&recipe.Calories, &recipe.Protein, &recipe.Fat, &recipe.Carbs, &recipe.Fiber, &recipe.Sodium)
	
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Recipe not found"})
		return
	}
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	// Parse JSON strings into slices
	if ingredientsJSON != "" {
		json.Unmarshal([]byte(ingredientsJSON), &recipe.Ingredients)
	}
	if instructionsJSON != "" {
		json.Unmarshal([]byte(instructionsJSON), &recipe.Instructions)
	}
	
	c.JSON(http.StatusOK, recipe)
}

func setupRoutes() *gin.Engine {
	r := gin.Default()
	
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	})
	
	// MCP Server endpoint
	r.POST("/mcp", handleMCPRequest)
	
	// Original API endpoints
	api := r.Group("/api")
	{
		api.GET("/recipes/search", searchRecipes)
		api.GET("/recipe/:id", getRecipeByID)
		api.GET("/diet-plans", getDietPlans)
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "healthy"})
		})
	}
	
	return r
}

func Handler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		initDB()
	}
	
	router := setupRoutes()
	router.ServeHTTP(w, r)
}

func main() {
	initDB()
	
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	r := setupRoutes()
}