package services

import (
	"errors"

	"github.com/tomimandalaputra/e-commerce-go/internal/dto"
	"github.com/tomimandalaputra/e-commerce-go/internal/models"
	"gorm.io/gorm"
)

type CartService struct {
	db *gorm.DB
}

func NewCartService(db *gorm.DB) *CartService {
	return &CartService{db: db}
}

func (s *CartService) GetCart(userID uint) (*dto.CartResponse, error) {
	var cart models.Cart
	err := s.db.Preload("CartItems.Product.Category").
		Preload("CartItems.Product.Images").
		Where("user_id = ?", userID).First(&cart).Error
	if err != nil {
		return nil, err
	}

	return s.convertToCartResponse(&cart), nil
}

func (s *CartService) AddToCart(userID uint, req *dto.AddToCartRequest) (*dto.CartResponse, error) {
	var cartResponse *dto.CartResponse

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Check if product exists
		var product models.Product
		if err := tx.First(&product, req.ProductID).Error; err != nil {
			return errors.New("product not found")
		}

		if product.Stock < req.Quantity {
			return errors.New("insufficient stock")
		}

		// Get or create cart
		var cart models.Cart
		if err := tx.Where("user_id = ?", userID).FirstOrCreate(&cart, models.Cart{UserID: userID}).Error; err != nil {
			return err
		}

		// Check if item already exists in cart (including soft-deleted)
		var cartItem models.CartItem
		if err := tx.Unscoped().Where("cart_id = ? AND product_id = ?", cart.ID, req.ProductID).First(&cartItem).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// Truly new item
				cartItem = models.CartItem{
					CartID:    cart.ID,
					ProductID: req.ProductID,
					Quantity:  req.Quantity,
				}
				if err := tx.Create(&cartItem).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		} else {
			// Item exists (either active or soft-deleted)
			if cartItem.DeletedAt.Valid {
				// Reactivate soft-deleted item
				cartItem.DeletedAt = gorm.DeletedAt{}
				cartItem.Quantity = req.Quantity
			} else {
				// Update quantity for active item
				cartItem.Quantity += req.Quantity
			}

			if cartItem.Quantity > product.Stock {
				return errors.New("insufficient stock")
			}

			if err := tx.Save(&cartItem).Error; err != nil {
				return err
			}
		}

		// Fetch updated cart with preloads
		var updatedCart models.Cart
		err := tx.Preload("CartItems.Product.Category").
			Preload("CartItems.Product.Images").
			Where("user_id = ?", userID).First(&updatedCart).Error
		if err != nil {
			return err
		}

		cartResponse = s.convertToCartResponse(&updatedCart)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return cartResponse, nil
}

func (s *CartService) UpdateCartItem(userID, itemID uint, req *dto.UpdateCartItemRequest) (*dto.CartResponse, error) {
	var cartResponse *dto.CartResponse

	err := s.db.Transaction(func(tx *gorm.DB) error {
		var cartItem models.CartItem
		if err := tx.Joins("JOIN carts ON cart_items.cart_id = carts.id").
			Where("cart_items.id = ? AND carts.user_id = ?", itemID, userID).
			First(&cartItem).Error; err != nil {
			return errors.New("cart item not found")
		}

		var product models.Product
		if err := tx.First(&product, cartItem.ProductID).Error; err != nil {
			return errors.New("product not found")
		}

		if product.Stock < req.Quantity {
			return errors.New("insufficient stock")
		}

		cartItem.Quantity = req.Quantity
		if err := tx.Save(&cartItem).Error; err != nil {
			return err
		}

		// Fetch updated cart with preloads
		var updatedCart models.Cart
		if err := tx.Preload("CartItems.Product.Category").
			Preload("CartItems.Product.Images").
			Where("user_id = ?", userID).First(&updatedCart).Error; err != nil {
			return err
		}

		cartResponse = s.convertToCartResponse(&updatedCart)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return cartResponse, nil
}

func (s *CartService) RemoveFromCart(userID, itemID uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Verify item belongs to user before deleting
		var cartItem models.CartItem
		if err := tx.Joins("JOIN carts ON cart_items.cart_id = carts.id").
			Where("cart_items.id = ? AND carts.user_id = ?", itemID, userID).
			First(&cartItem).Error; err != nil {
			return errors.New("cart item not found")
		}

		return tx.Delete(&cartItem).Error
	})
}

func (s *CartService) convertToCartResponse(cart *models.Cart) *dto.CartResponse {
	cartItems := make([]dto.CartItemResponse, len(cart.CartItems)) // memory allocation
	var total float64

	for i := range cart.CartItems {
		subtotal := float64(cart.CartItems[i].Quantity) * cart.CartItems[i].Product.Price
		total += subtotal

		images := make([]dto.ProductImageResponse, len(cart.CartItems[i].Product.Images))
		for j := range cart.CartItems[i].Product.Images {
			img := &cart.CartItems[i].Product.Images[j]
			images[j] = dto.ProductImageResponse{
				ID:        img.ID,
				URL:       img.URL,
				AltText:   img.AltText,
				IsPrimary: img.IsPrimary,
				CreatedAt: img.CreatedAt,
			}
		}

		cartItems[i] = dto.CartItemResponse{
			ID: cart.CartItems[i].ID,
			Product: dto.ProductResponse{
				ID:          cart.CartItems[i].Product.ID,
				CategoryID:  cart.CartItems[i].Product.CategoryID,
				Name:        cart.CartItems[i].Product.Name,
				Description: cart.CartItems[i].Product.Description,
				Price:       cart.CartItems[i].Product.Price,
				Stock:       cart.CartItems[i].Product.Stock,
				SKU:         cart.CartItems[i].Product.SKU,
				IsActive:    cart.CartItems[i].Product.IsActive,
				Category: dto.CategoryResponse{
					ID:          cart.CartItems[i].Product.Category.ID,
					Name:        cart.CartItems[i].Product.Category.Name,
					Description: cart.CartItems[i].Product.Category.Description,
					IsActive:    cart.CartItems[i].Product.Category.IsActive,
				},
				Images: images,
			},
			Quantity: cart.CartItems[i].Quantity,
			Subtotal: subtotal,
		}
	}

	return &dto.CartResponse{
		ID:        cart.ID,
		UserID:    cart.UserID,
		CartItems: cartItems,
		Total:     total,
	}
}
