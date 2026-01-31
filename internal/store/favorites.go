package store

import (
	"github.com/rjsadow/launchpad/internal/models"
)

// GetFavorites retrieves all favorites for a user
func (db *DB) GetFavorites(userID string) ([]models.Favorite, error) {
	rows, err := db.Query(`
		SELECT user_id, app_id, created_at FROM favorites WHERE user_id = ?
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var favorites []models.Favorite
	for rows.Next() {
		var f models.Favorite
		if err := rows.Scan(&f.UserID, &f.AppID, &f.CreatedAt); err != nil {
			return nil, err
		}
		favorites = append(favorites, f)
	}

	return favorites, rows.Err()
}

// GetFavoriteAppIDs retrieves just the app IDs of user's favorites
func (db *DB) GetFavoriteAppIDs(userID string) (map[string]bool, error) {
	rows, err := db.Query("SELECT app_id FROM favorites WHERE user_id = ?", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	favs := make(map[string]bool)
	for rows.Next() {
		var appID string
		if err := rows.Scan(&appID); err != nil {
			return nil, err
		}
		favs[appID] = true
	}

	return favs, rows.Err()
}

// AddFavorite adds an app to user's favorites
func (db *DB) AddFavorite(userID, appID string) error {
	_, err := db.Exec(`
		INSERT OR IGNORE INTO favorites (user_id, app_id) VALUES (?, ?)
	`, userID, appID)
	return err
}

// RemoveFavorite removes an app from user's favorites
func (db *DB) RemoveFavorite(userID, appID string) error {
	_, err := db.Exec("DELETE FROM favorites WHERE user_id = ? AND app_id = ?", userID, appID)
	return err
}

// IsFavorite checks if an app is in user's favorites
func (db *DB) IsFavorite(userID, appID string) (bool, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM favorites WHERE user_id = ? AND app_id = ?
	`, userID, appID).Scan(&count)
	return count > 0, err
}
