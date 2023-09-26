package main

import (
    "context"
    "encoding/json"
    "net/http"
    "os"
    "log"
    "fmt"

    "github.com/redis/go-redis/v9"
    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"
    "github.com/joho/godotenv"
    "github.com/google/uuid"
)

type Recipe struct {
    Title   string `json:"title"`
    Content string `json:"content"`
}

type SavedRecipe struct {
	Id     string `json:"id"`
    Recipe Recipe `json:"recipe"`

}

func goDotEnvVariable(key string) string {
    return os.Getenv(key)
}

func deleteRecipeByKey(ctx context.Context, key string) error {
    client := redis.NewClient(&redis.Options{
        Addr:     goDotEnvVariable("REDIS_ADDRESS"),
        Password: goDotEnvVariable("REDIS_PASSWORD"),
        DB:       0,
    })
    _, err := client.HDel(ctx, "recipes", key).Result()
    return err
}

func createRecipe(recipe Recipe, ctx context.Context) error {
    client := redis.NewClient(&redis.Options{
        Addr:     goDotEnvVariable("REDIS_ADDRESS"),
        Password: goDotEnvVariable("REDIS_PASSWORD"),
        DB:       0,
    })

    id := uuid.New()

    recipeJSON, err := json.Marshal(recipe)
    if err != nil {
        return err
    }

    err = client.HSet(ctx, "recipes", id.String(), recipeJSON).Err()
    if err != nil {
        return err
    }

    return nil
}

func findRecipeByKey(ctx context.Context, key string) (Recipe, error) {
    client := redis.NewClient(&redis.Options{
        Addr:     goDotEnvVariable("REDIS_ADDRESS"),
        Password: goDotEnvVariable("REDIS_PASSWORD"),
        DB:       0,
    })
    val, err := client.HGet(ctx, "recipes", key).Result()
    if err != nil {
        return Recipe{}, err
    }

    var recipe Recipe
    if err := json.Unmarshal([]byte(val), &recipe); err != nil {
        return Recipe{}, err
    }

    return recipe, nil
}

func updateRecipeByKey(ctx context.Context, key string, updatedRecipe Recipe) error {
    client := redis.NewClient(&redis.Options{
        Addr:     goDotEnvVariable("REDIS_ADDRESS"),
        Password: goDotEnvVariable("REDIS_PASSWORD"),
        DB:       0,
    })
    exists, err := client.HExists(ctx, "recipes", key).Result()
    if err != nil {
        return err
    }

    if !exists {
        return fmt.Errorf("Chave não encontrada: %s", key)
    }

    updatedRecipeJSON, err := json.Marshal(updatedRecipe)
    if err != nil {
        return err
    }

    err = client.HSet(ctx, "recipes", key, updatedRecipeJSON).Err()
    return err
}

func getRecipes(ctx context.Context) ([]SavedRecipe, error) {
    client := redis.NewClient(&redis.Options{
        Addr:     goDotEnvVariable("REDIS_ADDRESS"),
        Password: goDotEnvVariable("REDIS_PASSWORD"),
        DB:       0,
    })

    keys, err := client.HKeys(ctx, "recipes").Result()
    if err != nil {
        return nil, err
    }

    var recipes []SavedRecipe

    for _, key := range keys {
        val, err := client.HGet(ctx, "recipes", key).Result()
        if err != nil {
            return nil, err
        }

        var recipe Recipe
        if err := json.Unmarshal([]byte(val), &recipe); err != nil {
            return nil, err
        }

		recipeWithID := SavedRecipe{
            Id:     key,   // Use a chave como ID
            Recipe: recipe, // Use a receita deserializada
        }

        recipes = append(recipes, recipeWithID)
    }

    return recipes, nil
}

func main() {
    e := echo.New()

    httpPort := os.Getenv("PORT")
    if httpPort == "" {
        httpPort = "8088"
    }

    ctx := context.Background()

    e.Use(middleware.Logger())
    e.Use(middleware.Recover())

    e.GET("/", func(c echo.Context) error {
        return c.HTML(http.StatusOK, "Hello, World!")
    })

    e.GET("/health", func(c echo.Context) error {
        return c.JSON(http.StatusOK, struct{ Status string }{Status: "OK"})
    })

    e.GET("/recipes", func(c echo.Context) error {
        recipes, err := getRecipes(ctx)
        if err != nil {
            return err
        }

        return c.JSON(http.StatusOK, recipes)
    })

	e.GET("/recipes/:key", func(c echo.Context) error {
		key := c.Param("key")
	
		recipe, err := findRecipeByKey(ctx, key)
		if err != nil {
			return err
		}
	
		return c.JSON(http.StatusOK, recipe)
	})

    e.POST("/recipe", func(c echo.Context) error {
        var recipe Recipe
        if err := c.Bind(&recipe); err != nil {
            return err
        }

        err := createRecipe(recipe, ctx)
        if err != nil {
            return err
        }

        return c.JSON(http.StatusCreated, map[string]string{"message": "Receita criada com sucesso!"})
    })

	e.DELETE("/recipes/:key", func(c echo.Context) error {
		key := c.Param("key")
	
		err := deleteRecipeByKey(ctx, key)
		if err != nil {
			return err
		}
	
		return c.JSON(http.StatusOK, map[string]string{"message": "Receita deletada com sucesso!"})
	})

	e.PUT("/recipes/:key", func(c echo.Context) error {
		key := c.Param("key")
	
		var updatedRecipe Recipe
		if err := c.Bind(&updatedRecipe); err != nil {
			return err
		}
	
		err := updateRecipeByKey(ctx, key, updatedRecipe)
		if err != nil {
			return err
		}
	
		return c.JSON(http.StatusOK, map[string]string{"message": "Receita atualizada com sucesso!"})
	})

    e.Logger.Fatal(e.Start(":" + httpPort))
}
