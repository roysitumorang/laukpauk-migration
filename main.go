package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"
	"unsafe"
	"uuid"

	_ "github.com/go-sql-driver/mysql"
	"github.com/goccy/go-json"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/nyaruka/phonenumbers"
	"golang.org/x/crypto/bcrypt"
)

const (
	numbers             = "0123456789"
	lowerCasedAlphabets = "123456789abcdefghijkmnopqrstuvwxyz"
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
		OldReferencedBy  *int64
		ReferencedBy     *uuid.UUID
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

	LandingPageBlock struct {
		ID               uuid.UUID
		Name             string
		CustomAttributes map[string]string
		CreatedBy,
		UpdatedBy uuid.UUID
		OldCreatedBy,
		OldUpdatedBy int64
		CreatedAt,
		UpdatedAt time.Time
		Items []*LandingPageBlockItem
	}

	LandingPageBlockItem struct {
		Subject string
		Body,
		Photo,
		LinkText,
		LinkURL string
	}

	LandingPageBlockItemData struct {
		Subject string `json:"title"`
		Body    string `json:"content"`
		Photo   string `json:"img"`
	}

	LandingPageBlockItemHero struct {
		Subject  string `json:"heading"`
		Body     string `json:"content"`
		LinkURL  string `json:"buttonLink"`
		LinkText string `json:"buttonText"`
		Photo    string `json:"img"`
	}
)

func main() {
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	err := godotenv.Load(".env")
	if err != nil {
		Log(slog.LevelError, err.Error())
		return
	}
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
		if errRollback := tx.Rollback(ctx); errRollback != nil {
			Log(slog.LevelError, errRollback.Error())
		}
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
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
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
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
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
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
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
				, name
				, mobile_phone_confirmed_at
				, mobile_phone
				, username
				, email_confirmed_at
				, email
				, encrypted_password
				, created_by
				, created_at
				, updated_by
				, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			RETURNING id`,
			"admin",
			admin.Name,
			admin.UpdatedAt,
			admin.MobilePhone,
			admin.Username,
			admin.UpdatedAt,
			admin.Email,
			admin.Password,
			admin.CreatedBy,
			admin.CreatedAt,
			admin.UpdatedBy,
			admin.UpdatedAt,
		).Scan(&admin.ID); err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
			Log(slog.LevelError, err.Error())
			return
		}
		mapAdminIDs[admin.OldID] = admin.ID
		if _, err = tx.Exec(
			ctx,
			`INSERT INTO admins (
				user_id
				, admin_level
				, photo
			) VALUES ($1, $2, $3)`,
			admin.ID,
			admin.Role,
			admin.Photo,
		); err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
			Log(slog.LevelError, err.Error())
			return
		}
		admins[i] = admin
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
		if errRollback := tx.Rollback(ctx); errRollback != nil {
			Log(slog.LevelError, errRollback.Error())
		}
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
			&buyer.OldReferencedBy,
			&buyer.VerificationCode,
			&buyer.IsVerified,
			&buyer.OldCreatedBy,
			&buyer.CreatedAt,
			&buyer.OldUpdatedBy,
			&buyer.UpdatedAt,
		); err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
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
				if errRollback := tx.Rollback(ctx); errRollback != nil {
					Log(slog.LevelError, errRollback.Error())
				}
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
		if buyer.OldReferencedBy != nil && *buyer.OldReferencedBy == 0 {
			buyer.OldReferencedBy = nil
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
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
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
		if buyer.OldReferencedBy != nil {
			if referencedBy, ok := mapBuyerIDs[*buyer.OldReferencedBy]; ok {
				buyer.ReferencedBy = &referencedBy
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
				, name
				, confirmation_token
				, confirmed_at
				, mobile_phone
				, username
				, encrypted_password
				, created_by
				, created_at
				, updated_by
				, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			RETURNING id`,
			"buyer",
			buyer.Name,
			buyer.VerificationCode,
			confirmedAt,
			buyer.MobilePhone,
			buyer.Username,
			buyer.Password,
			buyer.CreatedBy,
			buyer.CreatedAt,
			buyer.UpdatedBy,
			buyer.UpdatedAt,
		).Scan(&buyer.ID); err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
			Log(slog.LevelError, err.Error())
			return
		}
		mapBuyerIDs[buyer.OldID] = buyer.ID
		if _, err = tx.Exec(
			ctx,
			`INSERT INTO buyers (
				user_id
				, referral_code
				, photo
				, address
				, referenced_by
			) VALUES ($1, $2, $3, $4, $5)`,
			buyer.ID,
			buyer.ReferralCode,
			buyer.Photo,
			buyer.Address,
			buyer.ReferencedBy,
		); err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
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
		if errRollback := tx.Rollback(ctx); errRollback != nil {
			Log(slog.LevelError, errRollback.Error())
		}
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
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
			Log(slog.LevelError, err.Error())
			return
		}
		articles = append(articles, article)
	}
	for i, article := range articles {
		if createdBy, ok := mapAdminIDs[article.OldCreatedBy]; ok {
			article.CreatedBy = createdBy
		}
		if updatedBy, ok := mapAdminIDs[article.OldUpdatedBy]; ok {
			article.UpdatedBy = updatedBy
		}
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
			1,
			article.Subject,
			article.Body,
			article.Photo,
			article.Link,
			article.CreatedBy,
			article.CreatedAt,
			article.UpdatedBy,
			article.UpdatedAt,
		).Scan(&article.ID); err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
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
			, updatedBy
			, updatedAt
		FROM blog
		ORDER BY id`,
	)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	if err != nil {
		if errRollback := tx.Rollback(ctx); errRollback != nil {
			Log(slog.LevelError, errRollback.Error())
		}
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
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
			Log(slog.LevelError, err.Error())
			return
		}
		blogs = append(blogs, blog)
	}
	for i, blog := range blogs {
		if createdBy, ok := mapAdminIDs[blog.OldCreatedBy]; ok {
			blog.CreatedBy = createdBy
		}
		if updatedBy, ok := mapAdminIDs[blog.OldUpdatedBy]; ok {
			blog.UpdatedBy = updatedBy
		}
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
			1,
			blog.Subject,
			blog.Body,
			blog.Slug,
			blog.Photo,
			blog.CreatedBy,
			blog.CreatedAt,
			blog.UpdatedBy,
			blog.UpdatedAt,
		).Scan(&blog.ID); err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
			Log(slog.LevelError, err.Error())
			return
		}
		blogs[i] = blog
	}
	// landing_page_blocks
	rows, err = mysql.QueryContext(
		ctx,
		`SELECT
			p.name
			, c.elementKey
			, c.elementValue
			, p.createdBy
			, p.createdAt
			, p.updatedBy
			, p.updatedAt
		FROM homepage p
		JOIN homepage_element c ON p.id = c.parent
		ORDER BY c.id`,
	)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	if err != nil {
		if errRollback := tx.Rollback(ctx); errRollback != nil {
			Log(slog.LevelError, errRollback.Error())
		}
		Log(slog.LevelError, err.Error())
		return
	}
	defer rows.Close()
	var landingPageBlocks []*LandingPageBlock
	mapLandingPageBlockNames := map[string]int{}
	for rows.Next() {
		var (
			landingPageBlockName,
			elementKey,
			elementValue string
			oldCreatedBy,
			oldUpdatedBy int64
			createdAt,
			updatedAt time.Time
		)
		if err = rows.Scan(
			&landingPageBlockName,
			&elementKey,
			&elementValue,
			&oldCreatedBy,
			&createdAt,
			&oldUpdatedBy,
			&updatedAt,
		); err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
			Log(slog.LevelError, err.Error())
			return
		}
		switch landingPageBlockName {
		case "fitur":
			landingPageBlockName = "feature"
		case "uang-belanja-gratis":
			landingPageBlockName = "free-shopping"
		case "komitmen":
			landingPageBlockName = "commitment"
		case "pahlawan-dapur":
			landingPageBlockName = "kitchen-hero"
		case "bantuan":
			landingPageBlockName = "faq"
		}
		index, ok := mapLandingPageBlockNames[landingPageBlockName]
		if !ok {
			landingPageBlocks = append(
				landingPageBlocks,
				new(LandingPageBlock{
					Name:             landingPageBlockName,
					CustomAttributes: map[string]string{},
					OldCreatedBy:     oldCreatedBy,
					CreatedAt:        createdAt,
					OldUpdatedBy:     oldUpdatedBy,
					UpdatedAt:        updatedAt,
				}),
			)
			index = len(landingPageBlocks) - 1
			mapLandingPageBlockNames[landingPageBlockName] = index
		}
		websiteBlock := landingPageBlocks[index]
		if strings.HasPrefix(elementValue, "{") {
			landingPageBlockItem := new(LandingPageBlockItem)
			switch landingPageBlockName {
			case "hero":
				var data LandingPageBlockItemHero
				if err = json.Unmarshal(String2ByteSlice(elementValue), &data); err == nil {
					landingPageBlockItem.Subject = data.Subject
					landingPageBlockItem.Body = data.Body
					landingPageBlockItem.Photo = data.Photo
					landingPageBlockItem.LinkText = data.LinkText
					landingPageBlockItem.LinkURL = data.LinkURL
				}
			case "feature", "why", "faq":
				var data LandingPageBlockItemData
				if err = json.Unmarshal(String2ByteSlice(elementValue), &data); err == nil {
					landingPageBlockItem.Subject = data.Subject
					landingPageBlockItem.Body = data.Body
					landingPageBlockItem.Photo = data.Photo
				}
			}
			websiteBlock.Items = append(websiteBlock.Items, landingPageBlockItem)
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
			websiteBlock.CustomAttributes[key] = elementValue
		}
		landingPageBlocks[index] = websiteBlock
	}
	for i, landingPageBlock := range landingPageBlocks {
		if createdBy, ok := mapAdminIDs[landingPageBlock.OldCreatedBy]; ok {
			landingPageBlock.CreatedBy = createdBy
		}
		if updatedBy, ok := mapAdminIDs[landingPageBlock.OldUpdatedBy]; ok {
			landingPageBlock.UpdatedBy = updatedBy
		}
		if err = tx.QueryRow(
			ctx,
			`INSERT INTO landing_page_blocks (
				record_status
				, name
				, custom_attributes
				, created_by
				, created_at
				, updated_by
				, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id`,
			1,
			landingPageBlock.Name,
			landingPageBlock.CustomAttributes,
			landingPageBlock.CreatedBy,
			landingPageBlock.CreatedAt,
			landingPageBlock.UpdatedBy,
			landingPageBlock.UpdatedAt,
		).Scan(&landingPageBlock.ID); err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
			Log(slog.LevelError, err.Error())
			return
		}
		for j, landingPageBlockItem := range landingPageBlock.Items {
			if _, err = tx.Exec(
				ctx,
				`INSERT INTO landing_page_blocks (
					record_status
					, parent_name
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
				1,
				landingPageBlock.Name,
				landingPageBlockItem.Subject,
				landingPageBlockItem.Body,
				landingPageBlockItem.LinkText,
				landingPageBlockItem.LinkURL,
				landingPageBlockItem.Photo,
				landingPageBlock.CreatedBy,
				landingPageBlock.CreatedAt,
				landingPageBlock.UpdatedBy,
				landingPageBlock.UpdatedAt,
			); err != nil {
				if errRollback := tx.Rollback(ctx); errRollback != nil {
					Log(slog.LevelError, errRollback.Error())
				}
				Log(slog.LevelError, err.Error())
				return
			}
			landingPageBlock.Items[j] = landingPageBlockItem
		}
		landingPageBlocks[i] = landingPageBlock
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
		args = append(
			args,
			"func", name,
			"file", fmt.Sprintf("%s:%d", file, line),
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
