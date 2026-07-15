package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
	"unsafe"
	"uuid"

	_ "github.com/go-sql-driver/mysql"
	"github.com/goccy/go-json"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/nyaruka/phonenumbers"
	"golang.org/x/crypto/bcrypt"
)

var (
	sb strings.Builder
)

const (
	RecordStatusActive  int = iota
	numbers                 = "0123456789"
	lowerCasedAlphabets     = "123456789abcdefghijkmnopqrstuvwxyz"
)
const (
	ProductTypeStandalone int = iota
	ProductTypeBundled
)

type (
	Admin struct {
		ID    uuid.UUID
		OldID int64
		Role,
		Photo,
		Name,
		Email,
		MobilePhone,
		Username,
		Password string
		CreatedBy,
		UpdatedBy *uuid.UUID
		OldCreatedBy,
		OldUpdatedBy *int64
		CreatedAt,
		UpdatedAt time.Time
	}
	Buyer struct {
		ID    uuid.UUID
		OldID int64
		ReferralCode,
		Photo,
		Name,
		Address,
		Username,
		Password string
		MobilePhone      *string
		OldReferrerID    *int64
		ReferrerID       *uuid.UUID
		VerificationCode *string
		IsVerified       int
		VerifiedAt       *time.Time
		CreatedBy,
		UpdatedBy *uuid.UUID
		OldCreatedBy,
		OldUpdatedBy *int64
		CreatedAt,
		UpdatedAt time.Time
	}
	Article struct {
		ID    uuid.UUID
		OldID int64
		Subject,
		Body,
		Photo,
		Link string
		CreatedBy,
		UpdatedBy uuid.UUID
		OldCreatedBy,
		OldUpdatedBy int64
		CreatedAt,
		UpdatedAt time.Time
	}
	Blog struct {
		ID    uuid.UUID
		OldID int64
		Subject,
		Body,
		Slug,
		Photo string
		CreatedBy,
		UpdatedBy uuid.UUID
		OldCreatedBy,
		OldUpdatedBy int64
		CreatedAt,
		UpdatedAt time.Time
	}
	Block struct {
		ID               uuid.UUID
		Name             string
		CustomAttributes map[string]string
		CreatedBy,
		UpdatedBy uuid.UUID
		OldCreatedBy,
		OldUpdatedBy int64
		CreatedAt,
		UpdatedAt time.Time
		Items []*BlockItem
	}
	BlockItem struct {
		Subject,
		Body string
		Photo,
		LinkText,
		LinkURL *string
	}
	BlockItemData struct {
		Subject string  `json:"title"`
		Body    string  `json:"content"`
		Photo   *string `json:"img"`
	}
	BlockItemHero struct {
		Subject  string  `json:"heading"`
		Body     string  `json:"content"`
		LinkURL  *string `json:"buttonLink"`
		LinkText *string `json:"buttonText"`
		Photo    *string `json:"img"`
	}
	ProductCategory struct {
		ID    uuid.UUID
		OldID int64
		Name,
		Description,
		Photo string
		OldCreatedBy,
		OldUpdatedBy int64
		CreatedBy,
		UpdatedBy uuid.UUID
		CreatedAt,
		UpdatedAt time.Time
	}
	Product struct {
		ID                   uuid.UUID
		OldID                int64
		ProductCategoryID    uuid.UUID
		OldProductCategoryID int64
		ProductType          int
		Name,
		Photo,
		Description string
		ReadyStock,
		Pinned bool
		BasePrice,
		Discount int64
		OldCreatedBy,
		OldUpdatedBy int64
		CreatedBy,
		UpdatedBy uuid.UUID
		CreatedAt,
		UpdatedAt time.Time
	}
	ProductVariant struct {
		ID                      uuid.UUID
		OldID                   int64
		ProductID               uuid.UUID
		OldProductID            int64
		StockUnit               string
		Price                   int64
		DiscountType            int
		DiscountRate            int64
		DiscountMinimumQuantity int64
		DiscountAmount          int64
		SalePrice               int64
		Description             string
		OldCreatedBy,
		OldUpdatedBy int64
		CreatedBy,
		UpdatedBy uuid.UUID
		CreatedAt,
		UpdatedAt time.Time
	}
	Setting struct {
		ID           uuid.UUID
		Key          string
		Value        string
		Description  string
		CreatedBy    uuid.UUID
		OldCreatedBy int64
		CreatedAt    time.Time
		UpdatedBy    uuid.UUID
		OldUpdatedBy int64
		UpdatedAt    time.Time
	}
)

func main() {
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
	}))
	slog.SetDefault(logger)
	err := godotenv.Load(".env")
	if err != nil {
		Log(slog.LevelError, err.Error())
		return
	}
	now := time.Now()
	regex := regexp.MustCompile(`[^A-Za-z0-9_.\-]`)
	var sb strings.Builder
	mysql, err := sql.Open("mysql", os.Getenv("MYSQL_DSN"))
	if err != nil {
		Log(slog.LevelError, err.Error())
		return
	}
	if err = mysql.PingContext(ctx); err != nil {
		Log(slog.LevelError, err.Error())
		return
	}
	config, err := pgxpool.ParseConfig(os.Getenv("PG_DSN"))
	if err != nil {
		Log(slog.LevelError, err.Error())
		return
	}
	pg, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		Log(slog.LevelError, err.Error())
		return
	}
	if err = pg.Ping(ctx); err != nil {
		Log(slog.LevelError, err.Error())
		return
	}
	temporaryPassword := String2ByteSlice(os.Getenv("TEMPORARY_PASSWORD"))
	tx, err := pg.Begin(ctx)
	if err != nil {
		Log(slog.LevelError, err.Error())
		return
	}
	defer func(ctx context.Context) {
		if err = tx.Rollback(ctx); errors.Is(err, pgx.ErrTxClosed) {
			err = nil
		}
		if err != nil {
			Log(slog.LevelError, err.Error())
		}
	}(ctx)
	// admins
	rows, err := mysql.QueryContext(
		ctx,
		`SELECT
			id
			, role
			, foto
			, nama
			, email
			, telepon
			, username
			, createdBy
			, createdAt
			, updatedBy
			, updatedAt
		FROM admin
		ORDER BY id`,
	)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	if err != nil {
		Log(slog.LevelError, err.Error())
		return
	}
	defer rows.Close()
	var admins []Admin
	mapUsernames := map[string]struct{}{}
	mapConfirmationTokens := map[string]struct{}{}
	mapMobilePhones := map[string]struct{}{}
	for rows.Next() {
		var admin Admin
		if err = rows.Scan(
			&admin.OldID,
			&admin.Role,
			&admin.Photo,
			&admin.Name,
			&admin.Email,
			&admin.MobilePhone,
			&admin.Username,
			&admin.OldCreatedBy,
			&admin.CreatedAt,
			&admin.OldUpdatedBy,
			&admin.UpdatedAt,
		); err != nil {
			Log(slog.LevelError, err.Error())
			return
		}
		if admin.Username == "" {
			admin.Username = RandomString(6)
		} else {
			admin.Username = strings.ToLower(regex.ReplaceAllString(admin.Username, ""))
		}
		for {
			if _, ok := mapUsernames[admin.Username]; !ok {
				mapUsernames[admin.Username] = struct{}{}
				break
			}
			admin.Username = RandomString(6)
		}
		mobilePhone, err := phonenumbers.Parse(admin.MobilePhone, "ID")
		if err != nil {
			Log(slog.LevelError, err.Error())
			return
		}
		admin.MobilePhone = phonenumbers.Format(mobilePhone, phonenumbers.E164)
		for {
			if _, ok := mapMobilePhones[admin.MobilePhone]; !ok {
				mapMobilePhones[admin.MobilePhone] = struct{}{}
				break
			}
			sb.Reset()
			_, _ = sb.WriteString(admin.MobilePhone)
			_, _ = sb.WriteString(RandomNumber(1))
			admin.MobilePhone = sb.String()
		}
		hashByte, err := bcrypt.GenerateFromPassword(temporaryPassword, bcrypt.MinCost)
		if err != nil {
			Log(slog.LevelError, err.Error())
			return
		}
		admin.Password = ByteSlice2String(hashByte)
		if admin.OldCreatedBy != nil && *admin.OldCreatedBy == 0 {
			admin.OldCreatedBy = nil
		}
		if admin.OldUpdatedBy != nil && *admin.OldUpdatedBy == 0 {
			admin.OldUpdatedBy = nil
		}
		admins = append(admins, admin)
	}
	var adminID uuid.UUID
	mapAdminIDs := map[int64]uuid.UUID{}
	for i, admin := range admins {
		if admin.OldCreatedBy != nil {
			if createdBy, ok := mapAdminIDs[*admin.OldCreatedBy]; ok {
				admin.CreatedBy = &createdBy
			}
		}
		if admin.OldUpdatedBy != nil {
			if updatedBy, ok := mapAdminIDs[*admin.OldUpdatedBy]; ok {
				admin.UpdatedBy = &updatedBy
			}
		}
		if err = tx.QueryRow(
			ctx,
			`INSERT INTO users (
				user_type
				, user_status
				, name
				, mobile_phone_confirmed_at
				, mobile_phone
				, username
				, email_confirmed_at
				, email
				, encrypted_password
				, photo
				, created_by
				, created_at
				, updated_by
				, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
			RETURNING id`,
			0,
			0,
			admin.Name,
			admin.UpdatedAt,
			admin.MobilePhone,
			admin.Username,
			admin.UpdatedAt,
			admin.Email,
			admin.Password,
			admin.Photo,
			admin.CreatedBy,
			admin.CreatedAt,
			admin.UpdatedBy,
			admin.UpdatedAt,
		).Scan(&admin.ID); err != nil {
			Log(slog.LevelError, err.Error())
			return
		}
		mapAdminIDs[admin.OldID] = admin.ID
		var adminType int
		switch admin.Role {
		case "superadmin":
			adminType = 1
		case "admin":
			fallthrough
		default:
			adminType = 0
		}
		if _, err = tx.Exec(
			ctx,
			`INSERT INTO admins (
				user_id
				, admin_type
			) VALUES ($1, $2)`,
			admin.ID,
			adminType,
		); err != nil {
			Log(slog.LevelError, err.Error())
			return
		}
		admins[i] = admin
		adminID = admin.ID
	}
	// buyers
	rows, err = mysql.QueryContext(
		ctx,
		`SELECT
			id
			, kodeReferal
			, foto
			, nama
			, alamat
			, nomorHP
			, username
			, referencedBy
			, verifyCode
			, isVerified
			, createdBy
			, createdAt
			, updatedBy
			, updatedAt
		FROM user
		ORDER BY id`,
	)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	if err != nil {
		Log(slog.LevelError, err.Error())
		return
	}
	defer rows.Close()
	var buyers []Buyer
	mapReferralCodes := map[string]struct{}{}
	for rows.Next() {
		var buyer Buyer
		if err = rows.Scan(
			&buyer.OldID,
			&buyer.ReferralCode,
			&buyer.Photo,
			&buyer.Name,
			&buyer.Address,
			&buyer.MobilePhone,
			&buyer.Username,
			&buyer.OldReferrerID,
			&buyer.VerificationCode,
			&buyer.IsVerified,
			&buyer.OldCreatedBy,
			&buyer.CreatedAt,
			&buyer.OldUpdatedBy,
			&buyer.UpdatedAt,
		); err != nil {
			Log(slog.LevelError, err.Error())
			return
		}
		if buyer.ReferralCode == "" {
			buyer.ReferralCode = RandomString(6)
		}
		buyer.ReferralCode = strings.ToUpper(buyer.ReferralCode)
		for {
			if _, ok := mapReferralCodes[buyer.ReferralCode]; !ok {
				mapReferralCodes[buyer.ReferralCode] = struct{}{}
				break
			}
			buyer.ReferralCode = strings.ToUpper(RandomString(6))
		}
		if buyer.Username == "" {
			buyer.Username = buyer.Name
		}
		buyer.Username = strings.ToLower(regex.ReplaceAllString(buyer.Username, ""))
		for {
			if _, ok := mapUsernames[buyer.Username]; !ok {
				mapUsernames[buyer.Username] = struct{}{}
				break
			}
			sb.Reset()
			_, _ = sb.WriteString(buyer.Username)
			_, _ = sb.WriteString(RandomNumber(2))
			buyer.Username = sb.String()
		}
		if buyer.MobilePhone != nil && *buyer.MobilePhone != "" {
			mobilePhone, err := phonenumbers.Parse(*buyer.MobilePhone, "ID")
			if err != nil {
				Log(slog.LevelError, err.Error(), "mobile_phone", buyer.MobilePhone)
				return
			}
			buyer.MobilePhone = new(phonenumbers.Format(mobilePhone, phonenumbers.E164))
			for {
				if _, ok := mapMobilePhones[*buyer.MobilePhone]; !ok {
					mapMobilePhones[*buyer.MobilePhone] = struct{}{}
					break
				}
				sb.Reset()
				_, _ = sb.WriteString(*buyer.MobilePhone)
				_, _ = sb.WriteString(RandomNumber(1))
				buyer.MobilePhone = new(sb.String())
			}
		} else {
			buyer.MobilePhone = nil
		}
		if buyer.OldReferrerID != nil && *buyer.OldReferrerID == 0 {
			buyer.OldReferrerID = nil
		}
		if buyer.VerificationCode != nil {
			if *buyer.VerificationCode == "" {
				buyer.VerificationCode = nil
			} else {
				for {
					if _, ok := mapConfirmationTokens[*buyer.VerificationCode]; !ok {
						mapConfirmationTokens[*buyer.VerificationCode] = struct{}{}
						break
					}
					buyer.VerificationCode = new(RandomNumber(6))
				}
			}
		}
		hashByte, err := bcrypt.GenerateFromPassword(temporaryPassword, bcrypt.MinCost)
		if err != nil {
			Log(slog.LevelError, err.Error())
			return
		}
		buyer.Password = ByteSlice2String(hashByte)
		if buyer.OldCreatedBy != nil && *buyer.OldCreatedBy == 0 {
			buyer.OldCreatedBy = nil
		}
		if buyer.OldUpdatedBy != nil && *buyer.OldUpdatedBy == 0 {
			buyer.OldUpdatedBy = nil
		}
		buyers = append(buyers, buyer)
	}
	mapBuyerIDs := map[int64]uuid.UUID{}
	for i, buyer := range buyers {
		if buyer.OldReferrerID != nil {
			if referencedBy, ok := mapBuyerIDs[*buyer.OldReferrerID]; ok {
				buyer.ReferrerID = &referencedBy
			}
		}
		if buyer.OldCreatedBy != nil {
			if createdBy, ok := mapBuyerIDs[*buyer.OldCreatedBy]; ok {
				buyer.CreatedBy = &createdBy
			}
		}
		if buyer.OldUpdatedBy != nil {
			if updatedBy, ok := mapBuyerIDs[*buyer.OldUpdatedBy]; ok {
				buyer.UpdatedBy = &updatedBy
			}
		}
		var confirmedAt *time.Time
		if buyer.IsVerified == 1 {
			buyer.VerificationCode = nil
			confirmedAt = &buyer.UpdatedAt
		}
		if err = tx.QueryRow(
			ctx,
			`INSERT INTO users (
				user_type
				, user_status
				, name
				, confirmation_token
				, confirmed_at
				, mobile_phone
				, username
				, encrypted_password
				, photo
				, created_by
				, created_at
				, updated_by
				, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
			RETURNING id`,
			1,
			0,
			buyer.Name,
			buyer.VerificationCode,
			confirmedAt,
			buyer.MobilePhone,
			buyer.Username,
			buyer.Password,
			buyer.Photo,
			buyer.CreatedBy,
			buyer.CreatedAt,
			buyer.UpdatedBy,
			buyer.UpdatedAt,
		).Scan(&buyer.ID); err != nil {
			Log(slog.LevelError, err.Error())
			return
		}
		mapBuyerIDs[buyer.OldID] = buyer.ID
		if _, err = tx.Exec(
			ctx,
			`INSERT INTO buyers (
				user_id
				, referral_code
				, address
				, referrer_id
			) VALUES ($1, $2, $3, $4)`,
			buyer.ID,
			buyer.ReferralCode,
			buyer.Address,
			buyer.ReferrerID,
		); err != nil {
			Log(slog.LevelError, err.Error())
			return
		}
		buyers[i] = buyer
	}
	// articles
	rows, err = mysql.QueryContext(
		ctx,
		`SELECT
			id
			, judul
			, isi
			, foto
			, link
			, createdBy
			, createdAt
			, updatedBy
			, updatedAt
		FROM artikel
		ORDER BY id`,
	)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	if err != nil {
		Log(slog.LevelError, err.Error())
		return
	}
	defer rows.Close()
	var articles []Article
	for rows.Next() {
		var article Article
		if err = rows.Scan(
			&article.OldID,
			&article.Subject,
			&article.Body,
			&article.Photo,
			&article.Link,
			&article.OldCreatedBy,
			&article.CreatedAt,
			&article.OldUpdatedBy,
			&article.UpdatedAt,
		); err != nil {
			Log(slog.LevelError, err.Error())
			return
		}
		articles = append(articles, article)
	}
	for i, article := range articles {
		createdBy, ok := mapAdminIDs[article.OldCreatedBy]
		if !ok {
			createdBy = adminID
		}
		article.CreatedBy = createdBy
		updatedBy, ok := mapAdminIDs[article.OldUpdatedBy]
		if !ok {
			updatedBy = adminID
		}
		article.UpdatedBy = updatedBy
		if err = tx.QueryRow(
			ctx,
			`INSERT INTO articles (
				record_status
				, subject
				, body
				, photo
				, link
				, created_by
				, created_at
				, updated_by
				, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING id`,
			RecordStatusActive,
			article.Subject,
			article.Body,
			article.Photo,
			article.Link,
			article.CreatedBy,
			article.CreatedAt,
			article.UpdatedBy,
			article.UpdatedAt,
		).Scan(&article.ID); err != nil {
			Log(slog.LevelError, err.Error())
			return
		}
		articles[i] = article
	}
	// blogs
	rows, err = mysql.QueryContext(
		ctx,
		`SELECT
			id
			, foto
			, title
			, content
			, permalink
			, createdBy
			, createdAt
			, COALESCE(updatedBy, createdBy)
			, COALESCE(updatedAt, createdAt)
		FROM blog
		ORDER BY id`,
	)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	if err != nil {
		Log(slog.LevelError, err.Error())
		return
	}
	defer rows.Close()
	var blogs []Blog
	for rows.Next() {
		var blog Blog
		if err = rows.Scan(
			&blog.OldID,
			&blog.Photo,
			&blog.Subject,
			&blog.Body,
			&blog.Slug,
			&blog.OldCreatedBy,
			&blog.CreatedAt,
			&blog.OldUpdatedBy,
			&blog.UpdatedAt,
		); err != nil {
			Log(slog.LevelError, err.Error())
			return
		}
		blogs = append(blogs, blog)
	}
	for i, blog := range blogs {
		createdBy, ok := mapAdminIDs[blog.OldCreatedBy]
		if !ok {
			createdBy = adminID
		}
		blog.CreatedBy = createdBy
		updatedBy, ok := mapAdminIDs[blog.OldUpdatedBy]
		if !ok {
			updatedBy = adminID
		}
		blog.UpdatedBy = updatedBy
		if err = tx.QueryRow(
			ctx,
			`INSERT INTO blogs (
				record_status
				, subject
				, body
				, slug
				, photo
				, created_by
				, created_at
				, updated_by
				, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING id`,
			RecordStatusActive,
			blog.Subject,
			blog.Body,
			blog.Slug,
			blog.Photo,
			blog.CreatedBy,
			blog.CreatedAt,
			blog.UpdatedBy,
			blog.UpdatedAt,
		).Scan(&blog.ID); err != nil {
			Log(slog.LevelError, err.Error())
			return
		}
		blogs[i] = blog
	}
	// blocks
	rows, err = mysql.QueryContext(
		ctx,
		`SELECT
			p.name
			, c.elementKey
			, c.elementValue
			, p.createdBy
			, p.createdAt
			, COALESCE(p.updatedBy, p.createdBy)
			, COALESCE(p.updatedAt, p.createdAt)
		FROM homepage p
		JOIN homepage_element c ON p.id = c.parent
		ORDER BY c.id`,
	)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	if err != nil {
		Log(slog.LevelError, err.Error())
		return
	}
	defer rows.Close()
	var blocks []*Block
	mapBlockNames := map[string]int{}
	for rows.Next() {
		var (
			blockName,
			elementKey,
			elementValue string
			oldCreatedBy,
			oldUpdatedBy int64
			createdAt,
			updatedAt time.Time
		)
		if err = rows.Scan(
			&blockName,
			&elementKey,
			&elementValue,
			&oldCreatedBy,
			&createdAt,
			&oldUpdatedBy,
			&updatedAt,
		); err != nil {
			Log(slog.LevelError, err.Error())
			return
		}
		switch blockName {
		case "fitur":
			blockName = "feature"
		case "uang-belanja-gratis":
			blockName = "free-shopping"
		case "komitmen":
			blockName = "commitment"
		case "pahlawan-dapur":
			blockName = "kitchen-hero"
		case "bantuan":
			blockName = "faq"
		}
		index, ok := mapBlockNames[blockName]
		if !ok {
			blocks = append(
				blocks,
				new(Block{
					Name:             blockName,
					CustomAttributes: map[string]string{},
					OldCreatedBy:     oldCreatedBy,
					CreatedAt:        createdAt,
					OldUpdatedBy:     oldUpdatedBy,
					UpdatedAt:        updatedAt,
				}),
			)
			index = len(blocks) - 1
			mapBlockNames[blockName] = index
		}
		block := blocks[index]
		if strings.HasPrefix(elementValue, "{") {
			blockItem := new(BlockItem)
			switch blockName {
			case "hero":
				var data BlockItemHero
				if err = json.Unmarshal(String2ByteSlice(elementValue), &data); err == nil {
					blockItem.Subject = data.Subject
					blockItem.Body = data.Body
					blockItem.Photo = data.Photo
					blockItem.LinkText = data.LinkText
					blockItem.LinkURL = data.LinkURL
				}
			case "feature", "why", "faq":
				var data BlockItemData
				if err = json.Unmarshal(String2ByteSlice(elementValue), &data); err == nil {
					blockItem.Subject = data.Subject
					blockItem.Body = data.Body
					blockItem.Photo = data.Photo
				}
			}
			block.Items = append(block.Items, blockItem)
		} else {
			var key string
			switch elementKey {
			case "_title":
				key = "subject"
			case "_buttonText":
				key = "link_text"
			case "_buttonLink":
				key = "link_url"
			case "_content":
				key = "body"
			case "_img":
				key = "photo"
			}
			block.CustomAttributes[key] = elementValue
		}
		blocks[index] = block
	}
	for i, block := range blocks {
		createdBy, ok := mapAdminIDs[block.OldCreatedBy]
		if !ok {
			createdBy = adminID
		}
		block.CreatedBy = createdBy
		updatedBy, ok := mapAdminIDs[block.OldUpdatedBy]
		if !ok {
			updatedBy = adminID
		}
		block.UpdatedBy = updatedBy
		if err = tx.QueryRow(
			ctx,
			`INSERT INTO blocks (
				record_status
				, name
				, custom_attributes
				, created_by
				, created_at
				, updated_by
				, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id`,
			RecordStatusActive,
			block.Name,
			block.CustomAttributes,
			block.CreatedBy,
			block.CreatedAt,
			block.UpdatedBy,
			block.UpdatedAt,
		).Scan(&block.ID); err != nil {
			Log(slog.LevelError, err.Error())
			return
		}
		for j, blockItem := range block.Items {
			if _, err = tx.Exec(
				ctx,
				`INSERT INTO block_items (
					record_status
					, block_name
					, subject
					, body
					, link_text
					, link_url
					, photo
					, created_by
					, created_at
					, updated_by
					, updated_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
				RecordStatusActive,
				block.Name,
				blockItem.Subject,
				blockItem.Body,
				blockItem.LinkText,
				blockItem.LinkURL,
				blockItem.Photo,
				block.CreatedBy,
				block.CreatedAt,
				block.UpdatedBy,
				block.UpdatedAt,
			); err != nil {
				Log(slog.LevelError, err.Error())
				return
			}
			block.Items[j] = blockItem
		}
		blocks[i] = block
	}
	// product_categories
	rows, err = mysql.QueryContext(
		ctx,
		`SELECT
			id
			, nama
			, keterangan
			, gambar
			, createdBy
			, createdAt
			, updatedBy
			, updatedAt
		FROM kategori_produk
		ORDER BY id`,
	)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	if err != nil {
		Log(slog.LevelError, err.Error())
		return
	}
	defer rows.Close()
	var productCategories []ProductCategory
	for rows.Next() {
		var productCategory ProductCategory
		if err = rows.Scan(
			&productCategory.OldID,
			&productCategory.Name,
			&productCategory.Description,
			&productCategory.Photo,
			&productCategory.OldCreatedBy,
			&productCategory.CreatedAt,
			&productCategory.OldUpdatedBy,
			&productCategory.UpdatedAt,
		); err != nil {
			Log(slog.LevelError, err.Error())
			return
		}
		createdBy, ok := mapAdminIDs[productCategory.OldCreatedBy]
		if !ok {
			createdBy = adminID
		}
		productCategory.CreatedBy = createdBy
		updatedBy, ok := mapAdminIDs[productCategory.OldUpdatedBy]
		if !ok {
			updatedBy = adminID
		}
		productCategory.UpdatedBy = updatedBy
		if productCategory.Photo != os.Getenv("PRODUCT_CATEGORY_DEFAULT_PHOTO") {
			if _, err = os.Stat(oldProductCategoryPhotoLocation(&sb, productCategory.Photo)); err != nil && errors.Is(err, os.ErrNotExist) {
				productCategory.Photo = os.Getenv("PRODUCT_CATEGORY_DEFAULT_PHOTO")
			}
		}
		productCategories = append(productCategories, productCategory)
	}
	productCategories = append(productCategories, ProductCategory{
		Name:      "Lain-Lain",
		Photo:     os.Getenv("PRODUCT_CATEGORY_DEFAULT_PHOTO"),
		CreatedBy: adminID,
		CreatedAt: now,
		UpdatedBy: adminID,
		UpdatedAt: now,
	})
	var defaultProductCategory ProductCategory
	var bundledProductCategoryID uuid.UUID
	mapProductCategoryIDs := map[int64]uuid.UUID{}
	for i, productCategory := range productCategories {
		if productCategory.Photo != os.Getenv("PRODUCT_CATEGORY_DEFAULT_PHOTO") {
			sb.Reset()
			_, _ = fmt.Fprintf(&sb, "%s%s", uuid.New().String(), filepath.Ext(productCategory.Photo))
			destination := sb.String()
			if err = copyFile(
				oldProductCategoryPhotoLocation(&sb, productCategory.Photo),
				newPhotoLocation(&sb, destination),
			); err != nil {
				Log(slog.LevelError, err.Error())
				return
			}
			productCategory.Photo = destination
		}
		if err = tx.QueryRow(
			ctx,
			`INSERT INTO product_categories (
				record_status
				, name
				, description
				, photo
				, created_by
				, created_at
				, updated_by
				, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id`,
			RecordStatusActive,
			productCategory.Name,
			productCategory.Description,
			productCategory.Photo,
			productCategory.CreatedBy,
			productCategory.CreatedAt,
			productCategory.UpdatedBy,
			productCategory.UpdatedAt,
		).Scan(&productCategory.ID); err != nil {
			Log(slog.LevelError, err.Error())
			return
		}
		productCategories[i] = productCategory
		defaultProductCategory = productCategory
		mapProductCategoryIDs[productCategory.OldID] = productCategory.ID
		if productCategory.Name == "Paket" {
			bundledProductCategoryID = productCategory.ID
		}
	}
	// standalone products
	rows, err = mysql.QueryContext(
		ctx,
		`SELECT
			a.id
			, a.foto
			, a.nama
			, a.idKategoriProduk
			, a.stokAvailable = 1
			, a.superior = 1
			, a.createdBy
			, a.createdAt
			, a.updatedBy
			, a.updatedAt
		FROM produk a
		WHERE EXISTS(
			SELECT 1
			FROM penjual b
			WHERE b.id = a.idPenjual
			AND b.namaToko = 'Laukpauk Inpres'
		)
		ORDER BY a.id`,
	)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	if err != nil {
		Log(slog.LevelError, err.Error())
		return
	}
	defer rows.Close()
	var products []Product
	mapProductNames := map[string]struct{}{}
	for rows.Next() {
		var product Product
		if err = rows.Scan(
			&product.OldID,
			&product.Photo,
			&product.Name,
			&product.OldProductCategoryID,
			&product.ReadyStock,
			&product.Pinned,
			&product.OldCreatedBy,
			&product.CreatedAt,
			&product.OldUpdatedBy,
			&product.UpdatedAt,
		); err != nil {
			Log(slog.LevelError, err.Error())
			return
		}
		product.ProductType = ProductTypeStandalone
		productName := strings.ToLower(product.Name)
		if _, ok := mapProductNames[productName]; ok {
			continue
		}
		productCategoryID, ok := mapProductCategoryIDs[product.OldProductCategoryID]
		if !ok {
			productCategoryID = defaultProductCategory.ID
		}
		product.ProductCategoryID = productCategoryID
		createdBy, ok := mapAdminIDs[product.OldCreatedBy]
		if !ok {
			createdBy = adminID
		}
		product.CreatedBy = createdBy
		updatedBy, ok := mapAdminIDs[product.OldUpdatedBy]
		if !ok {
			updatedBy = adminID
		}
		product.UpdatedBy = updatedBy
		if product.Photo != os.Getenv("PRODUCT_DEFAULT_PHOTO") {
			if _, err = os.Stat(oldProductPhotoLocation(&sb, product.Photo)); err != nil && errors.Is(err, os.ErrNotExist) {
				product.Photo = os.Getenv("PRODUCT_DEFAULT_PHOTO")
			}
		}
		products = append(products, product)
		mapProductNames[productName] = struct{}{}
	}
	mapProductIDs := map[int64]uuid.UUID{}
	var params []any
	for i, product := range products {
		if product.Photo != os.Getenv("PRODUCT_DEFAULT_PHOTO") {
			sb.Reset()
			_, _ = fmt.Fprintf(&sb, "%s%s", uuid.New().String(), filepath.Ext(product.Photo))
			destination := sb.String()
			if err = copyFile(
				oldProductPhotoLocation(&sb, product.Photo),
				newPhotoLocation(&sb, destination),
			); err != nil {
				Log(slog.LevelError, err.Error())
				return
			}
			product.Photo = destination
		}
		if err = tx.QueryRow(
			ctx,
			`INSERT INTO products (
				record_status
				, product_category_id
				, product_type
				, name
				, photo
				, description
				, ready_stock
				, pinned
				, created_by
				, created_at
				, updated_by
				, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			RETURNING id`,
			RecordStatusActive,
			product.ProductCategoryID,
			product.ProductType,
			product.Name,
			product.Photo,
			product.Description,
			product.ReadyStock,
			product.Pinned,
			product.CreatedBy,
			product.CreatedAt,
			product.UpdatedBy,
			product.UpdatedAt,
		).Scan(&product.ID); err != nil {
			Log(slog.LevelError, err.Error())
			return
		}
		products[i] = product
		mapProductIDs[product.OldID] = product.ID
		params = append(params, product.OldID)
	}
	sb.Reset()
	_, _ = sb.WriteString(
		`SELECT
			a.idProduk
			, a.nama
			, a.harga
			, a.diskon
			, a.keterangan
			, b.createdBy
			, b.createdAt
			, b.updatedBy
			, b.updatedAt
		FROM produk_harga a
		JOIN produk b ON a.idProduk = b.id
		WHERE a.idProduk IN (`,
	)
	for i := range params {
		if i > 0 {
			_, _ = sb.WriteString(",")
		}
		_, _ = sb.WriteString("?")
	}
	_, _ = sb.WriteString(") ORDER BY a.id")
	// product_variants
	rows, err = mysql.QueryContext(
		ctx,
		sb.String(),
		params...,
	)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	if err != nil {
		Log(slog.LevelError, err.Error())
		return
	}
	defer rows.Close()
	var productVariants []ProductVariant
	mapProductVariantStockUnits := map[uuid.UUID]map[string]struct{}{}
	for rows.Next() {
		var productVariant ProductVariant
		if err = rows.Scan(
			&productVariant.OldProductID,
			&productVariant.StockUnit,
			&productVariant.Price,
			&productVariant.DiscountRate,
			&productVariant.Description,
			&productVariant.OldCreatedBy,
			&productVariant.CreatedAt,
			&productVariant.OldUpdatedBy,
			&productVariant.UpdatedAt,
		); err != nil {
			Log(slog.LevelError, err.Error())
			return
		}
		productID, ok := mapProductIDs[productVariant.OldProductID]
		if !ok {
			continue
		}
		if _, ok := mapProductVariantStockUnits[productID][productVariant.StockUnit]; ok {
			continue
		}
		productVariant.ProductID = productID
		createdBy, ok := mapAdminIDs[productVariant.OldCreatedBy]
		if !ok {
			createdBy = adminID
		}
		productVariant.CreatedBy = createdBy
		updatedBy, ok := mapAdminIDs[productVariant.OldUpdatedBy]
		if !ok {
			updatedBy = adminID
		}
		productVariant.UpdatedBy = updatedBy
		productVariants = append(productVariants, productVariant)
		if _, ok := mapProductVariantStockUnits[productID]; !ok {
			mapProductVariantStockUnits[productID] = map[string]struct{}{}
		}
		mapProductVariantStockUnits[productID][productVariant.StockUnit] = struct{}{}
	}
	for i, productVariant := range productVariants {
		if err = tx.QueryRow(
			ctx,
			`INSERT INTO product_variants (
				record_status
				, product_id
				, stock_unit
				, price
				, discount_type
				, discount_rate
				, discount_minimum_quantity
				, discount_amount
				, sale_price
				, description
				, created_by
				, created_at
				, updated_by
				, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
			RETURNING id`,
			RecordStatusActive,
			productVariant.ProductID,
			productVariant.StockUnit,
			productVariant.Price,
			productVariant.DiscountType,
			productVariant.DiscountRate,
			productVariant.DiscountMinimumQuantity,
			productVariant.DiscountAmount,
			productVariant.SalePrice,
			productVariant.Description,
			productVariant.CreatedBy,
			productVariant.CreatedAt,
			productVariant.UpdatedBy,
			productVariant.UpdatedAt,
		).Scan(&productVariant.ID); err != nil {
			Log(slog.LevelError, err.Error())
			return
		}
		productVariants[i] = productVariant
	}
	// bundled products
	rows, err = mysql.QueryContext(
		ctx,
		`SELECT
			id
			, foto
			, nama
			, harga
			, diskon
			, stockAvailable = 1
			, superior = 1
			, keterangan
			, createdBy
			, createdAt
			, updatedBy
			, updatedAt
		FROM produk_paket
		ORDER BY id`,
	)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	if err != nil {
		Log(slog.LevelError, err.Error())
		return
	}
	defer rows.Close()
	products = []Product{}
	for rows.Next() {
		var product Product
		if err = rows.Scan(
			&product.OldID,
			&product.Photo,
			&product.Name,
			&product.BasePrice,
			&product.Discount,
			&product.ReadyStock,
			&product.Pinned,
			&product.Description,
			&product.OldCreatedBy,
			&product.CreatedAt,
			&product.OldUpdatedBy,
			&product.UpdatedAt,
		); err != nil {
			Log(slog.LevelError, err.Error())
			return
		}
		product.ProductType = ProductTypeBundled
		productName := strings.ToLower(product.Name)
		if _, ok := mapProductNames[productName]; ok {
			continue
		}
		product.ProductCategoryID = bundledProductCategoryID
		createdBy, ok := mapAdminIDs[product.OldCreatedBy]
		if !ok {
			createdBy = adminID
		}
		product.CreatedBy = createdBy
		updatedBy, ok := mapAdminIDs[product.OldUpdatedBy]
		if !ok {
			updatedBy = adminID
		}
		product.UpdatedBy = updatedBy
		if product.Photo != os.Getenv("BUNDLE_DEFAULT_PHOTO") {
			if _, err = os.Stat(oldProductPhotoLocation(&sb, product.Photo)); err != nil && errors.Is(err, os.ErrNotExist) {
				product.Photo = os.Getenv("BUNDLE_DEFAULT_PHOTO")
			}
		}
		products = append(products, product)
		mapProductNames[productName] = struct{}{}
	}
	for i, product := range products {
		if product.Photo != os.Getenv("BUNDLE_DEFAULT_PHOTO") {
			sb.Reset()
			_, _ = fmt.Fprintf(&sb, "%s%s", uuid.New().String(), filepath.Ext(product.Photo))
			destination := sb.String()
			if err = copyFile(
				oldProductPhotoLocation(&sb, product.Photo),
				newPhotoLocation(&sb, destination),
			); err != nil {
				Log(slog.LevelError, err.Error())
				return
			}
			product.Photo = destination
		}
		if err = tx.QueryRow(
			ctx,
			`INSERT INTO products (
				record_status
				, product_category_id
				, product_type
				, name
				, photo
				, description
				, ready_stock
				, pinned
				, created_by
				, created_at
				, updated_by
				, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			RETURNING id`,
			RecordStatusActive,
			product.ProductCategoryID,
			product.ProductType,
			product.Name,
			product.Photo,
			product.Description,
			product.ReadyStock,
			product.Pinned,
			product.CreatedBy,
			product.CreatedAt,
			product.UpdatedBy,
			product.UpdatedAt,
		).Scan(&product.ID); err != nil {
			Log(slog.LevelError, err.Error())
			return
		}
		if _, err = tx.Exec(
			ctx,
			`INSERT INTO product_variants (
				record_status
				, product_id
				, stock_unit
				, price
				, discount_type
				, discount_rate
				, discount_minimum_quantity
				, discount_amount
				, sale_price
				, description
				, created_by
				, created_at
				, updated_by
				, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
			RecordStatusActive,
			product.ID,
			"paket",
			product.BasePrice,
			0,
			product.Discount,
			0,
			product.Discount,
			product.BasePrice-product.Discount,
			"",
			product.CreatedBy,
			product.CreatedAt,
			product.UpdatedBy,
			product.UpdatedAt,
		); err != nil {
			Log(slog.LevelError, err.Error())
			return
		}
		products[i] = product
	}
	// settings
	rows, err = mysql.QueryContext(
		ctx,
		`SELECT
			kunci
			, nilai
			, keterangan
			, createdBy
			, createdAt
			, updatedBy
			, updatedAt
		FROM pengaturan
		ORDER BY createdAt`,
	)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	if err != nil {
		Log(slog.LevelError, err.Error())
		return
	}
	defer rows.Close()
	var settings []Setting
	for rows.Next() {
		var setting Setting
		if err = rows.Scan(
			&setting.Key,
			&setting.Value,
			&setting.Description,
			&setting.OldCreatedBy,
			&setting.CreatedAt,
			&setting.OldUpdatedBy,
			&setting.UpdatedAt,
		); err != nil {
			Log(slog.LevelError, err.Error())
			return
		}
		settings = append(settings, setting)
	}
	for i, setting := range settings {
		createdBy, ok := mapAdminIDs[setting.OldCreatedBy]
		if !ok {
			createdBy = adminID
		}
		setting.CreatedBy = createdBy
		updatedBy, ok := mapAdminIDs[setting.OldUpdatedBy]
		if !ok {
			updatedBy = adminID
		}
		setting.UpdatedBy = updatedBy
		if err = tx.QueryRow(
			ctx,
			`INSERT INTO settings (
				active
				, key
				, value
				, description
				, created_by
				, created_at
				, updated_by
				, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id`,
			true,
			setting.Key,
			setting.Value,
			setting.Description,
			setting.CreatedBy,
			setting.CreatedAt,
			setting.UpdatedBy,
			setting.UpdatedAt,
		).Scan(&setting.ID); err != nil {
			Log(slog.LevelError, err.Error())
			return
		}
		settings[i] = setting
	}
	// delivery_plans
	if _, err = tx.Exec(
		ctx,
		`INSERT INTO delivery_plans (
			record_status
			, name
			, transaction_window
			, same_day
			, same_day_window
			, same_day_interval
			, next_day
			, next_day_window
			, next_day_interval
			, next_two_day
			, next_two_day_window
			, next_two_day_interval
			, created_by
			, created_at
			, updated_by
			, updated_at
		) VALUES (
			$1
			, $2
			, tstzrange($3, $4, '[]')
			, $5
			, tstzrange($6, $7)
			, $8
			, $9
			, tstzrange($10, $11, '[]')
			, $12
			, $13
			, tstzrange($14, $15, '[]')
			, $16
			, $17
			, $18
			, $19
			, $20
		), (
			$21
			, $22
			, tstzrange($23, $24, '[]')
			, $25
			, tstzrange($26, $27)
			, $28
			, $29
			, tstzrange($30, $31, '[]')
			, $32
			, $33
			, tstzrange($34, $35, '[]')
			, $36
			, $37
			, $38
			, $39
			, $40
		)`,
		RecordStatusActive,
		"Estimasi pengantaran untuk transaksi sebelum jam 22:00",
		"0001-01-01 00:00:00+07:07:12 BC",
		"0001-01-01 21:59:59+07:07:12 BC",
		false,
		nil,
		nil,
		nil,
		true,
		"0001-01-02 08:00:00+07:07:12 BC",
		"0001-01-02 12:00:00+07:07:12 BC",
		"2h",
		true,
		"0001-01-03 08:00:00+07:07:12 BC",
		"0001-01-03 12:00:00+07:07:12 BC",
		"2h",
		adminID,
		now,
		adminID,
		now,
		RecordStatusActive,
		"Estimasi pengantaran untuk transaksi mulai jam 22:00",
		"0001-01-01 22:00:00+07:07:12 BC",
		"0001-01-01 23:59:59+07:07:12 BC",
		false,
		nil,
		nil,
		nil,
		true,
		"0001-01-02 10:00:00+07:07:12 BC",
		"0001-01-02 12:00:00+07:07:12 BC",
		"2h",
		true,
		"0001-01-03 08:00:00+07:07:12 BC",
		"0001-01-03 12:00:00+07:07:12 BC",
		"2h",
		adminID,
		now,
		adminID,
		now,
	); err != nil {
		Log(slog.LevelError, err.Error())
		return
	}
	if err = tx.Commit(ctx); err != nil {
		Log(slog.LevelError, err.Error())
	}
}

func Log(level slog.Level, msg string, args ...any) {
	switch level {
	case slog.LevelDebug:
		slog.Debug(msg, args...)
	case slog.LevelInfo:
		slog.Info(msg, args...)
	case slog.LevelWarn:
		slog.Warn(msg, args...)
	case slog.LevelError:
		var name string
		pc, file, line, _ := runtime.Caller(1)
		if fn := runtime.FuncForPC(pc); fn != nil {
			name = fn.Name()
		}
		sb.Reset()
		_, _ = fmt.Fprintf(&sb, "%s:%d", file, line)
		args = append(
			args,
			"func", name,
			"file", sb.String(),
			"line", line,
		)
		slog.Error(msg, args...)
	}
}

func String2ByteSlice(str string) []byte {
	return unsafe.Slice(unsafe.StringData(str), len(str))
}

func ByteSlice2String(bs []byte) string {
	return *(*string)(unsafe.Pointer(&bs))
}

func RandomString(length int) string {
	randomBytes := make([]byte, length)
	for {
		if _, err := rand.Read(randomBytes); err == nil {
			break
		}
	}
	for i := range length {
		randomBytes[i] = lowerCasedAlphabets[randomBytes[i]%34]
	}
	return ByteSlice2String(randomBytes)
}

func RandomNumber(length int) string {
	randomBytes := make([]byte, length)
	for {
		if _, err := rand.Read(randomBytes); err == nil {
			break
		}
	}
	for i := range length {
		randomBytes[i] = numbers[randomBytes[i]%10]
	}
	return ByteSlice2String(randomBytes)
}

func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()
	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()
	if _, err = io.Copy(destination, source); err != nil {
		return err
	}
	if err = destination.Sync(); err != nil {
		return err
	}
	return os.Chmod(dst, 0644)
}

func oldProductCategoryPhotoLocation(sb *strings.Builder, filename string) string {
	sb.Reset()
	_, _ = fmt.Fprintf(sb, "%s/product_category/%s", os.Getenv("PHOTO_DIRECTORY"), filename)
	return sb.String()
}

func oldProductPhotoLocation(sb *strings.Builder, filename string) string {
	sb.Reset()
	_, _ = fmt.Fprintf(sb, "%s/product/%s", os.Getenv("PHOTO_DIRECTORY"), filename)
	return sb.String()
}

func newPhotoLocation(sb *strings.Builder, filename string) string {
	sb.Reset()
	_, _ = fmt.Fprintf(sb, "%s/%s", os.Getenv("PHOTO_DIRECTORY"), filename)
	return sb.String()
}
