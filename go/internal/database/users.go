package database

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"-"`
}

func CreateUser(username, password string) error {
	_, err := DB.Exec("INSERT INTO users (username, password) VALUES (?, ?)", username, password)
	return err
}

func GetUserByUsername(username string) (*User, error) {
	row := DB.QueryRow("SELECT id, username, password FROM users WHERE username = ?", username)
	var user User
	err := row.Scan(&user.ID, &user.Username, &user.Password)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func GetUserByID(id int) (*User, error) {
	var user User
	err := DB.QueryRow("SELECT id, username FROM users WHERE id = ?", id).Scan(&user.ID, &user.Username)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func SearchUsers(query string) ([]User, error) {
	rows, err := DB.Query("SELECT id, username FROM users WHERE username LIKE ? LIMIT 5", query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Username); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}
