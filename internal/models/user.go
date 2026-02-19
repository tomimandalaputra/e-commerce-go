package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        int            `json:"id" grom:"primaryKey"`
	Email     string         `json:"email" grom:"uniqueIndex;not null"`
	Password  string         `json:"-" grom:"not null"`
	FirstName string         `json:"first_name" grom:"not null"`
	LastName  string         `json:"last_name" grom:"not null"`
	Phone     string         `json:"phone" grom:"not null"`
	IsActive  bool           `json:"is_active" grom:"default:true"`
	Role      UserRole       `json:"role" grom:"default:customer"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" grom:"index"`

	// Relationships
	RefreshTokens []RefreshToken `json:"-"`
	Orders        []Order        `json:"-"`
	Cart          Cart           `json:"-"`
}

type UserRole string

const (
	UserRoleAdmin    UserRole = "admin"
	UserRoleCustomer UserRole = "customer"
)

type RefreshToken struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	UserID    uint           `json:"user_id" gorm:"not null"`
	Token     string         `json:"token" gorm:"uniqueIndex;not null"`
	ExpiresAt time.Time      `json:"expires_at" gorm:"not null"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// Relationships
	User User `json:"-"`
}
