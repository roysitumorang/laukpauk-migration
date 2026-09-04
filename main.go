package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json/v2"
	"errors"
	"fmt"
	"image/jpeg"
	"image/png"
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
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/kolesa-team/go-webp/encoder"
	"github.com/kolesa-team/go-webp/webp"
	"github.com/nyaruka/phonenumbers"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	sb     strings.Builder
	photos []string
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
		Username string
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
	LandingPage struct {
		ID uuid.UUID
		OldName,
		Name string
		CustomAttributes map[string]string
		CreatedBy,
		UpdatedBy uuid.UUID
		OldCreatedBy,
		OldUpdatedBy int64
		CreatedAt,
		UpdatedAt *time.Time
		Items []*LandingPageItem
	}
	LandingPageItem struct {
		OldLandingPageName,
		Subject,
		Body string
		Photo,
		LinkText,
		LinkURL *string
	}
	LandingPageItemData struct {
		Subject string  `json:"title"`
		Body    string  `json:"content"`
		Photo   *string `json:"img"`
	}
	LandingPageItemHero struct {
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
		ProductCategoryName  string
		OldProductCategoryID int64
		ProductType          int
		Name,
		Photo,
		Description string
		ReadyStock,
		Pinned bool
		Hits,
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
		ProductCategoryID       uuid.UUID
		ProductCategoryName     string
		ProductID               uuid.UUID
		OldProductID            int64
		ProductName             string
		ProductReadyStock       bool
		StockUnit               string
		BasePrice               int64
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
	Banner struct {
		ID    uuid.UUID
		OldID int64
		Name,
		Description,
		Photo string
		Visible      bool
		CreatedBy    uuid.UUID
		OldCreatedBy int64
		CreatedAt    time.Time
		UpdatedBy    uuid.UUID
		OldUpdatedBy int64
		UpdatedAt    time.Time
	}
	RecipeGuide struct {
		OldRecipeID int64
		Instruction string
	}
	RecipeItem struct {
		OldRecipeID  int64
		ProductID    uuid.UUID
		OldProductID int64
		Dose,
		Note string
	}
	Recipe struct {
		ID    uuid.UUID
		OldID int64
		Name,
		Photo,
		Description,
		InstagramLink,
		FacebookLink string
		CreatedBy    uuid.UUID
		OldCreatedBy int64
		CreatedAt    time.Time
		UpdatedBy    uuid.UUID
		OldUpdatedBy int64
		UpdatedAt    time.Time
		Guides       []RecipeGuide
		Items        []RecipeItem
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
	tx, err := pg.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable,
	})
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
	sb.Reset()
	_, _ = fmt.Fprintf(&sb, "%s/admin/admin-default.png", os.Getenv("PHOTO_DIRECTORY"))
	src := sb.String()
	sb.Reset()
	_, _ = fmt.Fprintf(&sb, "%s/admin-default.png", os.Getenv("PHOTO_DIRECTORY"))
	dst := sb.String()
	photos = append(photos, dst)
	if err = copyFile(src, dst); err != nil {
		Log(slog.LevelError, err.Error())
		return
	}
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
	defer func() {
		_ = rows.Close()
	}()
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
	mapUserIDs := map[int64]uuid.UUID{}
	for i, admin := range admins {
		if admin.Email == "benni.surbakti@gmail.com" {
			if err = tx.QueryRow(
				ctx,
				`SELECT id
				FROM users
				WHERE email = $1`,
				admin.Email,
			).Scan(&admin.ID); err != nil {
				Log(slog.LevelError, err.Error())
				return
			}
			adminID = admin.ID
		} else {
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
					, created_at
					, updated_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
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
				admin.CreatedAt,
				admin.UpdatedAt,
			).Scan(&admin.ID); err != nil {
				Log(slog.LevelError, err.Error())
				return
			}
			var adminLevel int
			switch admin.Role {
			case "superadmin":
				adminLevel = 0
			case "admin":
				fallthrough
			default:
				adminLevel = 1
			}
			if _, err = tx.Exec(
				ctx,
				`INSERT INTO admins (
					id
					, admin_level
				) VALUES ($1, $2)
				RETURNING id`,
				admin.ID,
				adminLevel,
			); err != nil {
				Log(slog.LevelError, err.Error())
				return
			}
		}
		mapAdminIDs[admin.OldID] = admin.ID
		mapUserIDs[admin.OldID] = admin.ID
		admins[i] = admin
	}
	for i, admin := range admins {
		if admin.OldCreatedBy != nil {
			createdBy, ok := mapUserIDs[*admin.OldCreatedBy]
			if !ok {
				createdBy = adminID
			}
			admin.CreatedBy = &createdBy
		}
		if admin.OldUpdatedBy != nil {
			updatedBy, ok := mapUserIDs[*admin.OldUpdatedBy]
			if !ok {
				updatedBy = adminID
			}
			admin.UpdatedBy = &updatedBy
		}
		if _, err = tx.Exec(
			ctx,
			`UPDATE users SET
				created_by = $1
				, updated_by = $2
			WHERE id = $3`,
			admin.CreatedBy,
			admin.UpdatedBy,
			admin.ID,
		); err != nil {
			Log(slog.LevelError, err.Error())
			return
		}
		admins[i] = admin
	}
	var companyID uuid.UUID
	if err = tx.QueryRow(
		ctx,
		`SELECT id FROM companies`,
	).Scan(&companyID); err != nil {
		Log(slog.LevelError, err.Error())
		return
	}
	// buyers
	sb.Reset()
	_, _ = fmt.Fprintf(&sb, "%s/user/user-default.png", os.Getenv("PHOTO_DIRECTORY"))
	src = sb.String()
	sb.Reset()
	_, _ = fmt.Fprintf(&sb, "%s/buyer-default.png", os.Getenv("PHOTO_DIRECTORY"))
	dst = sb.String()
	photos = append(photos, dst)
	if err = copyFile(src, dst); err != nil {
		Log(slog.LevelError, err.Error())
		return
	}
	sb.Reset()
	_, _ = fmt.Fprintf(&sb, "%s/seller-default.png", os.Getenv("PHOTO_DIRECTORY"))
	dst = sb.String()
	photos = append(photos, dst)
	if err = copyFile(src, dst); err != nil {
		Log(slog.LevelError, err.Error())
		return
	}
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
		WHERE nama NOT LIKE '%laukpauk%'
			AND nama NOT LIKE '%toko%'
		ORDER BY id`,
	)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	if err != nil {
		Log(slog.LevelError, err.Error())
		return
	}
	defer func() {
		_ = rows.Close()
	}()
	var buyers []Buyer
	mapReferralCodes := map[string]struct{}{}
	caser := cases.Title(language.Und, cases.NoLower)
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
		buyer.Name = caser.String(buyer.Name)
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
		if buyer.OldCreatedBy != nil && *buyer.OldCreatedBy == 0 {
			buyer.OldCreatedBy = nil
		}
		if buyer.OldUpdatedBy != nil && *buyer.OldUpdatedBy == 0 {
			buyer.OldUpdatedBy = nil
		}
		buyers = append(buyers, buyer)
	}
	mapBuyers := map[int64]Buyer{}
	for i, buyer := range buyers {
		var confirmedAt *time.Time
		if buyer.IsVerified == 1 {
			buyer.VerificationCode = nil
			confirmedAt = &buyer.UpdatedAt
		}
		if buyer.Photo == "user-default.png" {
			buyer.Photo = "buyer-default.png"
		} else {
			sb.Reset()
			_, _ = fmt.Fprintf(&sb, "%s/user/%s", os.Getenv("PHOTO_DIRECTORY"), buyer.Photo)
			src = sb.String()
			sb.Reset()
			_, _ = fmt.Fprintf(&sb, "%s.webp", uuid.NewV7().String())
			buyer.Photo = sb.String()
			sb.Reset()
			_, _ = fmt.Fprintf(&sb, "%s/%s", os.Getenv("PHOTO_DIRECTORY"), buyer.Photo)
			dst = sb.String()
			photos = append(photos, dst)
			switch filepath.Ext(src) {
			case ".jpg", ".jpeg":
				if err = jpeg2webp(src, dst); err != nil {
					Log(slog.LevelError, err.Error())
					return
				}
			case ".png":
				if err = png2webp(src, dst); err != nil {
					Log(slog.LevelError, err.Error())
					return
				}
			case ".webp":
				if err = copyFile(src, dst); err != nil {
					Log(slog.LevelError, err.Error())
					return
				}
			}
		}
		if err = tx.QueryRow(
			ctx,
			`INSERT INTO users (
				company_id
				, user_type
				, user_status
				, name
				, mobile_phone_confirmation_token
				, mobile_phone_confirmation_sent_at
				, mobile_phone_confirmed_at
				, mobile_phone
				, username
				, photo
				, created_at
				, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			RETURNING id`,
			companyID,
			2,
			0,
			buyer.Name,
			buyer.VerificationCode,
			buyer.CreatedAt,
			confirmedAt,
			buyer.MobilePhone,
			buyer.Username,
			buyer.Photo,
			buyer.CreatedAt,
			buyer.UpdatedAt,
		).Scan(&buyer.ID); err != nil {
			Log(slog.LevelError, err.Error())
			return
		}
		if _, err = tx.Exec(
			ctx,
			`INSERT INTO buyers (
				id
				, referral_code
				, address
			) VALUES ($1, $2, $3)`,
			buyer.ID,
			buyer.ReferralCode,
			buyer.Address,
		); err != nil {
			Log(slog.LevelError, err.Error())
			return
		}
		buyers[i] = buyer
		mapBuyers[buyer.OldID] = buyer
	}
	for i, buyer := range buyers {
		if buyer.OldCreatedBy != nil {
			createdBy, ok := mapUserIDs[*buyer.OldCreatedBy]
			if !ok {
				createdBy = adminID
			}
			buyer.CreatedBy = &createdBy
		}
		if buyer.OldUpdatedBy != nil {
			updatedBy, ok := mapUserIDs[*buyer.OldUpdatedBy]
			if !ok {
				updatedBy = adminID
			}
			buyer.UpdatedBy = &updatedBy
		}
		if buyer.OldReferrerID != nil {
			if user, ok := mapBuyers[*buyer.OldReferrerID]; ok {
				buyer.ReferrerID = &user.ID
			}
		}
		if buyer.CreatedBy != nil || buyer.UpdatedBy != nil {
			if _, err = tx.Exec(
				ctx,
				`UPDATE users SET
					created_by = $1
					, updated_by = $2
				WHERE id = $3`,
				buyer.CreatedBy,
				buyer.UpdatedBy,
				buyer.ID,
			); err != nil {
				Log(slog.LevelError, err.Error())
				return
			}
		}
		if buyer.ReferrerID != nil {
			if _, err = tx.Exec(
				ctx,
				`UPDATE buyers SET
					referrer_id = $1
				WHERE id = $2`,
				buyer.ReferrerID,
				buyer.ID,
			); err != nil {
				Log(slog.LevelError, err.Error())
				return
			}
		}
		buyers[i] = buyer
	}
	// articles
	sb.Reset()
	_, _ = fmt.Fprintf(&sb, "%s/artikel/artikel-default.png", os.Getenv("PHOTO_DIRECTORY"))
	src = sb.String()
	sb.Reset()
	_, _ = fmt.Fprintf(&sb, "%s/article-default.png", os.Getenv("PHOTO_DIRECTORY"))
	dst = sb.String()
	photos = append(photos, dst)
	if err = copyFile(src, dst); err != nil {
		Log(slog.LevelError, err.Error())
		return
	}
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
	defer func() {
		_ = rows.Close()
	}()
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
		if article.Photo == "artikel-default.png" {
			article.Photo = "article-default.png"
		} else {
			sb.Reset()
			_, _ = fmt.Fprintf(&sb, "%s/artikel/%s", os.Getenv("PHOTO_DIRECTORY"), article.Photo)
			src = sb.String()
			sb.Reset()
			_, _ = fmt.Fprintf(&sb, "%s.webp", uuid.NewV7().String())
			article.Photo = sb.String()
			sb.Reset()
			_, _ = fmt.Fprintf(&sb, "%s/%s", os.Getenv("PHOTO_DIRECTORY"), article.Photo)
			dst = sb.String()
			photos = append(photos, dst)
			switch filepath.Ext(src) {
			case ".jpg", ".jpeg":
				if err = jpeg2webp(src, dst); err != nil {
					Log(slog.LevelError, err.Error())
					return
				}
			case ".png":
				if err = png2webp(src, dst); err != nil {
					Log(slog.LevelError, err.Error())
					return
				}
			case ".webp":
				if err = copyFile(src, dst); err != nil {
					Log(slog.LevelError, err.Error())
					return
				}
			}
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
	defer func() {
		_ = rows.Close()
	}()
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
		sb.Reset()
		_, _ = fmt.Fprintf(&sb, "%s/blog/%s", os.Getenv("PHOTO_DIRECTORY"), blog.Photo)
		src = sb.String()
		sb.Reset()
		_, _ = fmt.Fprintf(&sb, "%s.webp", uuid.NewV7().String())
		blog.Photo = sb.String()
		sb.Reset()
		_, _ = fmt.Fprintf(&sb, "%s/%s", os.Getenv("PHOTO_DIRECTORY"), blog.Photo)
		dst = sb.String()
		photos = append(photos, dst)
		switch filepath.Ext(src) {
		case ".jpg", ".jpeg":
			if err = jpeg2webp(src, dst); err != nil {
				Log(slog.LevelError, err.Error())
				return
			}
		case ".png":
			if err = png2webp(src, dst); err != nil {
				Log(slog.LevelError, err.Error())
				return
			}
		case ".webp":
			if err = copyFile(src, dst); err != nil {
				Log(slog.LevelError, err.Error())
				return
			}
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
	// landing_pages
	sb.Reset()
	_, _ = fmt.Fprintf(&sb, "%s/landing-page-imgs/pahlawanDapur", os.Getenv("PHOTO_DIRECTORY"))
	src = sb.String()
	sb.Reset()
	_, _ = fmt.Fprintf(&sb, "%s/landing-page-imgs/pahlawan-dapur", os.Getenv("PHOTO_DIRECTORY"))
	dst = sb.String()
	if _, err = os.Stat(src); err == nil {
		if err = os.Rename(src, dst); err != nil {
			Log(slog.LevelError, err.Error())
			return
		}
	}
	sb.Reset()
	_, _ = fmt.Fprintf(&sb, "%s/landing-page-imgs/uangBelanjaGratis", os.Getenv("PHOTO_DIRECTORY"))
	src = sb.String()
	sb.Reset()
	_, _ = fmt.Fprintf(&sb, "%s/landing-page-imgs/uang-belanja-gratis", os.Getenv("PHOTO_DIRECTORY"))
	dst = sb.String()
	if _, err = os.Stat(src); err == nil {
		if err = os.Rename(src, dst); err != nil {
			Log(slog.LevelError, err.Error())
			return
		}
	}
	rows, err = mysql.QueryContext(
		ctx,
		`SELECT
			COALESCE(p.name, CASE WHEN c.parent = '_halamanBantuan' THEN c.parent END)
			, c.elementKey
			, c.elementValue
			, COALESCE(p.createdBy, 0)
			, p.createdAt
			, COALESCE(p.updatedBy, 0)
			, COALESCE(p.updatedAt, p.createdAt)
		FROM homepage p
		RIGHT JOIN homepage_element c ON p.id = CASE WHEN c.parent = '_halamanBantuan' THEN NULL ELSE c.parent END
		WHERE c.parent != '0'
		ORDER BY c.id`,
	)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	if err != nil {
		Log(slog.LevelError, err.Error())
		return
	}
	defer func() {
		_ = rows.Close()
	}()
	var landingPages []*LandingPage
	mapLandingPageNameIndexes := map[string]int{}
	mapLandingPages := map[string]string{}
	var maxCreatedAt, maxUpdatedAt time.Time
	for rows.Next() {
		var (
			oldLandingPageName,
			elementKey,
			elementValue string
			oldCreatedBy,
			oldUpdatedBy int64
			createdAt,
			updatedAt *time.Time
		)
		if err = rows.Scan(
			&oldLandingPageName,
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
		landingPageName := oldLandingPageName
		switch oldLandingPageName {
		case "hero":
			landingPageName = "hero-slider"
		case "fitur":
			landingPageName = "feature"
		case "uang-belanja-gratis":
			landingPageName = "free-shopping"
		case "komitmen":
			landingPageName = "commitment"
		case "pahlawan-dapur":
			landingPageName = "kitchen-hero"
		case "_halamanBantuan":
			landingPageName = "faq"
		}
		if createdAt != nil {
			maxCreatedAt = *createdAt
		} else {
			createdAt = &maxCreatedAt
		}
		if updatedAt != nil {
			maxUpdatedAt = *updatedAt
		} else {
			updatedAt = &maxUpdatedAt
		}
		mapLandingPages[landingPageName] = oldLandingPageName
		index, ok := mapLandingPageNameIndexes[landingPageName]
		if !ok {
			landingPages = append(
				landingPages,
				&LandingPage{
					OldName:          mapLandingPages[landingPageName],
					Name:             landingPageName,
					CustomAttributes: map[string]string{},
					OldCreatedBy:     oldCreatedBy,
					CreatedAt:        createdAt,
					OldUpdatedBy:     oldUpdatedBy,
					UpdatedAt:        updatedAt,
				},
			)
			index = len(landingPages) - 1
			mapLandingPageNameIndexes[landingPageName] = index
		}
		landingPage := landingPages[index]
		if strings.HasPrefix(elementValue, "{") {
			landingPageItem := &LandingPageItem{
				OldLandingPageName: landingPage.OldName,
			}
			switch landingPageName {
			case "hero-slider":
				var data LandingPageItemHero
				if err = json.Unmarshal(String2ByteSlice(elementValue), &data); err == nil {
					landingPageItem.Subject = data.Subject
					landingPageItem.Body = data.Body
					landingPageItem.Photo = data.Photo
					landingPageItem.LinkText = data.LinkText
					landingPageItem.LinkURL = data.LinkURL
				}
			case "feature", "why", "faq":
				var data LandingPageItemData
				if err = json.Unmarshal(String2ByteSlice(elementValue), &data); err == nil {
					landingPageItem.Subject = data.Subject
					landingPageItem.Body = data.Body
					landingPageItem.Photo = data.Photo
				}
			}
			landingPage.Items = append(landingPage.Items, landingPageItem)
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
			landingPage.CustomAttributes[key] = elementValue
		}
		landingPages[index] = landingPage
	}
	for i, landingPage := range landingPages {
		createdBy, ok := mapAdminIDs[landingPage.OldCreatedBy]
		if !ok {
			createdBy = adminID
		}
		landingPage.CreatedBy = createdBy
		updatedBy, ok := mapAdminIDs[landingPage.OldUpdatedBy]
		if !ok {
			updatedBy = adminID
		}
		landingPage.UpdatedBy = updatedBy
		if photo, ok := landingPage.CustomAttributes["photo"]; ok {
			sb.Reset()
			_, _ = fmt.Fprintf(&sb, "%s/landing-page-imgs/%s/%s", os.Getenv("PHOTO_DIRECTORY"), landingPage.OldName, photo)
			src = sb.String()
			sb.Reset()
			_, _ = fmt.Fprintf(&sb, "%s.webp", uuid.NewV7().String())
			photo = sb.String()
			sb.Reset()
			_, _ = fmt.Fprintf(&sb, "%s/%s", os.Getenv("PHOTO_DIRECTORY"), photo)
			dst = sb.String()
			photos = append(photos, dst)
			switch filepath.Ext(src) {
			case ".jpg", ".jpeg":
				if err = jpeg2webp(src, dst); err != nil {
					Log(slog.LevelError, err.Error())
					return
				}
			case ".png":
				if err = png2webp(src, dst); err != nil {
					Log(slog.LevelError, err.Error())
					return
				}
			case ".webp":
				if err = copyFile(src, dst); err != nil {
					Log(slog.LevelError, err.Error())
					return
				}
			}
			landingPage.CustomAttributes["photo"] = photo
		}
		if err = tx.QueryRow(
			ctx,
			`INSERT INTO landing_pages (
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
			landingPage.Name,
			landingPage.CustomAttributes,
			landingPage.CreatedBy,
			landingPage.CreatedAt,
			landingPage.UpdatedBy,
			landingPage.UpdatedAt,
		).Scan(&landingPage.ID); err != nil {
			Log(slog.LevelError, err.Error())
			return
		}
		for j, landingPageItem := range landingPage.Items {
			if landingPageItem.Photo != nil {
				sb.Reset()
				_, _ = fmt.Fprintf(&sb, "%s/landing-page-imgs/%s/%s", os.Getenv("PHOTO_DIRECTORY"), landingPage.OldName, *landingPageItem.Photo)
				src = sb.String()
				sb.Reset()
				_, _ = fmt.Fprintf(&sb, "%s.webp", uuid.NewV7().String())
				landingPageItem.Photo = new(sb.String())
				sb.Reset()
				_, _ = fmt.Fprintf(&sb, "%s/%s", os.Getenv("PHOTO_DIRECTORY"), *landingPageItem.Photo)
				dst = sb.String()
				photos = append(photos, dst)
				switch filepath.Ext(src) {
				case ".jpg", ".jpeg":
					if err = jpeg2webp(src, dst); err != nil {
						Log(slog.LevelError, err.Error())
						return
					}
				case ".png":
					if err = png2webp(src, dst); err != nil {
						Log(slog.LevelError, err.Error())
						return
					}
				case ".webp":
					if err = copyFile(src, dst); err != nil {
						Log(slog.LevelError, err.Error())
						return
					}
				}
			}
			if _, err = tx.Exec(
				ctx,
				`INSERT INTO landing_page_items (
					record_status
					, landing_page_name
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
				landingPage.Name,
				landingPageItem.Subject,
				landingPageItem.Body,
				landingPageItem.LinkText,
				landingPageItem.LinkURL,
				landingPageItem.Photo,
				landingPage.CreatedBy,
				landingPage.CreatedAt,
				landingPage.UpdatedBy,
				landingPage.UpdatedAt,
			); err != nil {
				Log(slog.LevelError, err.Error())
				return
			}
			landingPage.Items[j] = landingPageItem
		}
		landingPages[i] = landingPage
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
	defer func() {
		_ = rows.Close()
	}()
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
		if productCategory.Photo == "kategori-produk-default.png" {
			productCategory.Photo = "product-category-default.png"
		} else {
			sb.Reset()
			_, _ = fmt.Fprintf(&sb, "%s/kategoriProduk/%s", os.Getenv("PHOTO_DIRECTORY"), productCategory.Photo)
			if _, err = os.Stat(sb.String()); err != nil && errors.Is(err, os.ErrNotExist) {
				productCategory.Photo = "product-category-default.png"
			}
		}
		productCategories = append(productCategories, productCategory)
	}
	productCategories = append(productCategories, ProductCategory{
		Name:      "Lain-Lain",
		Photo:     "product-category-default.png",
		CreatedBy: adminID,
		CreatedAt: now,
		UpdatedBy: adminID,
		UpdatedAt: now,
	})
	sb.Reset()
	_, _ = fmt.Fprintf(&sb, "%s/kategoriProduk/kategori-produk-default.png", os.Getenv("PHOTO_DIRECTORY"))
	src = sb.String()
	sb.Reset()
	_, _ = fmt.Fprintf(&sb, "%s/product-category-default.png", os.Getenv("PHOTO_DIRECTORY"))
	dst = sb.String()
	photos = append(photos, dst)
	if err = copyFile(src, dst); err != nil {
		Log(slog.LevelError, err.Error())
		return
	}
	var defaultProductCategory ProductCategory
	var bundledProductCategory ProductCategory
	mapProductCategories := map[int64]ProductCategory{}
	for i, productCategory := range productCategories {
		if productCategory.Photo != "product-category-default.png" {
			sb.Reset()
			_, _ = fmt.Fprintf(&sb, "%s/kategoriProduk/%s", os.Getenv("PHOTO_DIRECTORY"), productCategory.Photo)
			src = sb.String()
			sb.Reset()
			_, _ = fmt.Fprintf(&sb, "%s.webp", uuid.NewV7().String())
			productCategory.Photo = sb.String()
			sb.Reset()
			_, _ = fmt.Fprintf(&sb, "%s/%s", os.Getenv("PHOTO_DIRECTORY"), productCategory.Photo)
			dst = sb.String()
			photos = append(photos, dst)
			switch filepath.Ext(src) {
			case ".jpg", ".jpeg":
				if err = jpeg2webp(src, dst); err != nil {
					Log(slog.LevelError, err.Error())
					return
				}
			case ".png":
				if err = png2webp(src, dst); err != nil {
					Log(slog.LevelError, err.Error())
					return
				}
			case ".webp":
				if err = copyFile(src, dst); err != nil {
					Log(slog.LevelError, err.Error())
					return
				}
			}
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
		mapProductCategories[productCategory.OldID] = productCategory
		if productCategory.Name == "Paket" {
			bundledProductCategory = productCategory
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
	defer func() {
		_ = rows.Close()
	}()
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
		productCategory, ok := mapProductCategories[product.OldProductCategoryID]
		if !ok {
			productCategory = defaultProductCategory
		}
		product.ProductCategoryID = productCategory.ID
		product.ProductCategoryName = productCategory.Name
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
		if product.Photo == "produk-default.png" {
			product.Photo = "product-default.png"
		} else {
			sb.Reset()
			_, _ = fmt.Fprintf(&sb, "%s/produk/%s", os.Getenv("PHOTO_DIRECTORY"), product.Photo)
			if _, err = os.Stat(sb.String()); err != nil && errors.Is(err, os.ErrNotExist) {
				product.Photo = "product-default.png"
			}
		}
		products = append(products, product)
		mapProductNames[productName] = struct{}{}
	}
	sb.Reset()
	_, _ = fmt.Fprintf(&sb, "%s/produk/produk-default.png", os.Getenv("PHOTO_DIRECTORY"))
	src = sb.String()
	sb.Reset()
	_, _ = fmt.Fprintf(&sb, "%s/product-default.png", os.Getenv("PHOTO_DIRECTORY"))
	dst = sb.String()
	photos = append(photos, dst)
	if err = copyFile(src, dst); err != nil {
		Log(slog.LevelError, err.Error())
		return
	}
	mapProducts := map[int64]Product{}
	var params []any
	for i, product := range products {
		if product.Photo != "product-default.png" {
			sb.Reset()
			_, _ = fmt.Fprintf(&sb, "%s/produk/%s", os.Getenv("PHOTO_DIRECTORY"), product.Photo)
			src = sb.String()
			sb.Reset()
			_, _ = fmt.Fprintf(&sb, "%s.webp", uuid.NewV7().String())
			product.Photo = sb.String()
			sb.Reset()
			_, _ = fmt.Fprintf(&sb, "%s/%s", os.Getenv("PHOTO_DIRECTORY"), product.Photo)
			dst = sb.String()
			photos = append(photos, dst)
			switch filepath.Ext(src) {
			case ".jpg", ".jpeg":
				if err = jpeg2webp(src, dst); err != nil {
					Log(slog.LevelError, err.Error())
					return
				}
			case ".png":
				if err = png2webp(src, dst); err != nil {
					Log(slog.LevelError, err.Error())
					return
				}
			case ".webp":
				if err = copyFile(src, dst); err != nil {
					Log(slog.LevelError, err.Error())
					return
				}
			}
		}
		if err = tx.QueryRow(
			ctx,
			`INSERT INTO products (
				record_status
				, product_category_id
				, product_category_name
				, product_type
				, name
				, photo
				, description
				, ready_stock
				, pinned
				, hits
				, created_by
				, created_at
				, updated_by
				, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
			RETURNING id`,
			RecordStatusActive,
			product.ProductCategoryID,
			product.ProductCategoryName,
			product.ProductType,
			product.Name,
			product.Photo,
			product.Description,
			product.ReadyStock,
			product.Pinned,
			product.Hits,
			product.CreatedBy,
			product.CreatedAt,
			product.UpdatedBy,
			product.UpdatedAt,
		).Scan(&product.ID); err != nil {
			Log(slog.LevelError, err.Error())
			return
		}
		products[i] = product
		mapProducts[product.OldID] = product
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
	defer func() {
		_ = rows.Close()
	}()
	var productVariants []ProductVariant
	mapProductVariantStockUnits := map[uuid.UUID]map[string]struct{}{}
	mapProductLowestSellingPrices := map[uuid.UUID]int64{}
	for rows.Next() {
		var productVariant ProductVariant
		if err = rows.Scan(
			&productVariant.OldProductID,
			&productVariant.StockUnit,
			&productVariant.BasePrice,
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
		product, ok := mapProducts[productVariant.OldProductID]
		if !ok {
			continue
		}
		if _, ok := mapProductVariantStockUnits[product.ID][productVariant.StockUnit]; ok {
			continue
		}
		productVariant.ProductCategoryID = product.ProductCategoryID
		productVariant.ProductCategoryName = product.ProductCategoryName
		productVariant.ProductID = product.ID
		productVariant.ProductName = product.Name
		productVariant.ProductReadyStock = product.ReadyStock
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
		if _, ok := mapProductVariantStockUnits[product.ID]; !ok {
			mapProductVariantStockUnits[product.ID] = map[string]struct{}{}
		}
		mapProductVariantStockUnits[product.ID][productVariant.StockUnit] = struct{}{}
		lowestSellingPrice, ok := mapProductLowestSellingPrices[product.ID]
		if !ok || (lowestSellingPrice > 0 && productVariant.BasePrice < lowestSellingPrice) {
			lowestSellingPrice = productVariant.BasePrice
		}
		mapProductLowestSellingPrices[product.ID] = lowestSellingPrice
	}
	for i, productVariant := range productVariants {
		if err = tx.QueryRow(
			ctx,
			`INSERT INTO product_variants (
				record_status
				, product_category_id
				, product_category_name
				, product_id
				, product_name
				, product_ready_stock
				, stock_unit
				, base_price
				, selling_price
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
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
			RETURNING id`,
			RecordStatusActive,
			productVariant.ProductCategoryID,
			productVariant.ProductCategoryName,
			productVariant.ProductID,
			productVariant.ProductName,
			productVariant.ProductReadyStock,
			productVariant.StockUnit,
			productVariant.BasePrice,
			productVariant.BasePrice,
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
	for productID, lowestSellingPrice := range mapProductLowestSellingPrices {
		if _, err = tx.Exec(
			ctx,
			"UPDATE products SET lowest_selling_price = $1 WHERE id = $2",
			lowestSellingPrice,
			productID,
		); err != nil {
			Log(slog.LevelError, err.Error())
			return
		}
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
	defer func() {
		_ = rows.Close()
	}()
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
		product.ProductCategoryID = bundledProductCategory.ID
		product.ProductCategoryName = bundledProductCategory.Name
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
		if product.Photo == "paket-default.png" {
			product.Photo = "bundle-default.png"
		} else {
			sb.Reset()
			_, _ = fmt.Fprintf(&sb, "%s/paket/%s", os.Getenv("PHOTO_DIRECTORY"), product.Photo)
			if _, err = os.Stat(sb.String()); err != nil && errors.Is(err, os.ErrNotExist) {
				product.Photo = "bundle-default.png"
			}
		}
		products = append(products, product)
		mapProductNames[productName] = struct{}{}
	}
	sb.Reset()
	_, _ = fmt.Fprintf(&sb, "%s/paket/paket-default.png", os.Getenv("PHOTO_DIRECTORY"))
	src = sb.String()
	sb.Reset()
	_, _ = fmt.Fprintf(&sb, "%s/bundle-default.png", os.Getenv("PHOTO_DIRECTORY"))
	dst = sb.String()
	photos = append(photos, dst)
	if err = copyFile(src, dst); err != nil {
		Log(slog.LevelError, err.Error())
		return
	}
	for i, product := range products {
		if product.Photo != "bundle-default.png" {
			sb.Reset()
			_, _ = fmt.Fprintf(&sb, "%s/paket/%s", os.Getenv("PHOTO_DIRECTORY"), product.Photo)
			src = sb.String()
			sb.Reset()
			_, _ = fmt.Fprintf(&sb, "%s.webp", uuid.NewV7().String())
			product.Photo = sb.String()
			sb.Reset()
			_, _ = fmt.Fprintf(&sb, "%s/%s", os.Getenv("PHOTO_DIRECTORY"), product.Photo)
			dst = sb.String()
			photos = append(photos, dst)
			switch filepath.Ext(src) {
			case ".jpg", ".jpeg":
				if err = jpeg2webp(src, dst); err != nil {
					Log(slog.LevelError, err.Error())
					return
				}
			case ".png":
				if err = png2webp(src, dst); err != nil {
					Log(slog.LevelError, err.Error())
					return
				}
			case ".webp":
				if err = copyFile(src, dst); err != nil {
					Log(slog.LevelError, err.Error())
					return
				}
			}
		}
		if err = tx.QueryRow(
			ctx,
			`INSERT INTO products (
				record_status
				, product_category_id
				, product_category_name
				, product_type
				, name
				, photo
				, description
				, ready_stock
				, pinned
				, hits
				, lowest_selling_price
				, created_by
				, created_at
				, updated_by
				, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
			RETURNING id`,
			RecordStatusActive,
			product.ProductCategoryID,
			product.ProductCategoryName,
			product.ProductType,
			product.Name,
			product.Photo,
			product.Description,
			product.ReadyStock,
			product.Pinned,
			product.Hits,
			product.BasePrice,
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
				, product_category_id
				, product_category_name
				, product_id
				, product_name
				, product_ready_stock
				, stock_unit
				, base_price
				, selling_price
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
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)`,
			RecordStatusActive,
			product.ProductCategoryID,
			product.ProductCategoryName,
			product.ID,
			product.Name,
			product.ReadyStock,
			"paket",
			product.BasePrice,
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
	// banners
	rows, err = mysql.QueryContext(
		ctx,
		`SELECT
			id
			, nama
			, keterangan
			, gambar
			, shown = 1
			, createdBy
			, createdAt
			, updatedBy
			, updatedAt
		FROM banner
		ORDER BY id`,
	)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	if err != nil {
		Log(slog.LevelError, err.Error())
		return
	}
	defer func() {
		_ = rows.Close()
	}()
	var banners []Banner
	for rows.Next() {
		var banner Banner
		if err = rows.Scan(
			&banner.OldID,
			&banner.Name,
			&banner.Description,
			&banner.Photo,
			&banner.Visible,
			&banner.OldCreatedBy,
			&banner.CreatedAt,
			&banner.OldUpdatedBy,
			&banner.UpdatedAt,
		); err != nil {
			Log(slog.LevelError, err.Error())
			return
		}
		banners = append(banners, banner)
	}
	for i, banner := range banners {
		createdBy, ok := mapAdminIDs[banner.OldCreatedBy]
		if !ok {
			createdBy = adminID
		}
		banner.CreatedBy = createdBy
		updatedBy, ok := mapAdminIDs[banner.OldUpdatedBy]
		if !ok {
			updatedBy = adminID
		}
		banner.UpdatedBy = updatedBy
		sb.Reset()
		_, _ = fmt.Fprintf(&sb, "%s/banner/%s", os.Getenv("PHOTO_DIRECTORY"), banner.Photo)
		src = sb.String()
		sb.Reset()
		_, _ = fmt.Fprintf(&sb, "%s.webp", uuid.NewV7().String())
		banner.Photo = sb.String()
		sb.Reset()
		_, _ = fmt.Fprintf(&sb, "%s/%s", os.Getenv("PHOTO_DIRECTORY"), banner.Photo)
		dst = sb.String()
		photos = append(photos, dst)
		switch filepath.Ext(src) {
		case ".jpg", ".jpeg":
			if err = jpeg2webp(src, dst); err != nil {
				Log(slog.LevelError, err.Error())
				return
			}
		case ".png":
			if err = png2webp(src, dst); err != nil {
				Log(slog.LevelError, err.Error())
				return
			}
		case ".webp":
			if err = copyFile(src, dst); err != nil {
				Log(slog.LevelError, err.Error())
				return
			}
		}
		if err = tx.QueryRow(
			ctx,
			`INSERT INTO banners (
				record_status
				, name
				, description
				, photo
				, visible
				, created_by
				, created_at
				, updated_by
				, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING id`,
			RecordStatusActive,
			banner.Name,
			banner.Description,
			banner.Photo,
			banner.Visible,
			banner.CreatedBy,
			banner.CreatedAt,
			banner.UpdatedBy,
			banner.UpdatedAt,
		).Scan(&banner.ID); err != nil {
			Log(slog.LevelError, err.Error())
			return
		}
		banners[i] = banner
	}
	// recipes
	sb.Reset()
	_, _ = fmt.Fprintf(&sb, "%s/resep/resep-default.png", os.Getenv("PHOTO_DIRECTORY"))
	src = sb.String()
	sb.Reset()
	_, _ = fmt.Fprintf(&sb, "%s/recipe-default.png", os.Getenv("PHOTO_DIRECTORY"))
	dst = sb.String()
	photos = append(photos, dst)
	if err = copyFile(src, dst); err != nil {
		Log(slog.LevelError, err.Error())
		return
	}
	rows, err = mysql.QueryContext(
		ctx,
		`SELECT
			id
			, foto
			, nama
			, keterangan
			, igLink
			, fbLink
			, createdBy
			, createdAt
			, updatedBy
			, updatedAt
		FROM resep
		ORDER BY id`,
	)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	if err != nil {
		Log(slog.LevelError, err.Error())
		return
	}
	defer func() {
		_ = rows.Close()
	}()
	var recipes []Recipe
	mapRecipeOldIDs := map[int64]int{}
	for rows.Next() {
		var recipe Recipe
		if err = rows.Scan(
			&recipe.OldID,
			&recipe.Photo,
			&recipe.Name,
			&recipe.Description,
			&recipe.InstagramLink,
			&recipe.FacebookLink,
			&recipe.OldCreatedBy,
			&recipe.CreatedAt,
			&recipe.OldUpdatedBy,
			&recipe.UpdatedAt,
		); err != nil {
			Log(slog.LevelError, err.Error())
			return
		}
		recipes = append(recipes, recipe)
		mapRecipeOldIDs[recipe.OldID] = len(recipes) - 1
	}
	rows, err = mysql.QueryContext(
		ctx,
		`SELECT
			idResep
			, langkah
		FROM resep_cara_buat
		ORDER BY id`,
	)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	if err != nil {
		Log(slog.LevelError, err.Error())
		return
	}
	defer func() {
		_ = rows.Close()
	}()
	for rows.Next() {
		var recipeGuide RecipeGuide
		if err = rows.Scan(
			&recipeGuide.OldRecipeID,
			&recipeGuide.Instruction,
		); err != nil {
			Log(slog.LevelError, err.Error())
			return
		}
		if i, ok := mapRecipeOldIDs[recipeGuide.OldRecipeID]; ok {
			recipes[i].Guides = append(recipes[i].Guides, recipeGuide)
		}
	}
	rows, err = mysql.QueryContext(
		ctx,
		`SELECT
			idResep
			, idProduk
			, takaran
			, catatan
		FROM resep_item
		ORDER BY id`,
	)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	if err != nil {
		Log(slog.LevelError, err.Error())
		return
	}
	defer func() {
		_ = rows.Close()
	}()
	for rows.Next() {
		var recipeItem RecipeItem
		if err = rows.Scan(
			&recipeItem.OldRecipeID,
			&recipeItem.OldProductID,
			&recipeItem.Dose,
			&recipeItem.Note,
		); err != nil {
			Log(slog.LevelError, err.Error())
			return
		}
		if i, ok := mapRecipeOldIDs[recipeItem.OldRecipeID]; ok {
			recipes[i].Items = append(recipes[i].Items, recipeItem)
		}
	}
	for i, recipe := range recipes {
		createdBy, ok := mapAdminIDs[recipe.OldCreatedBy]
		if !ok {
			createdBy = adminID
		}
		recipe.CreatedBy = createdBy
		updatedBy, ok := mapAdminIDs[recipe.OldUpdatedBy]
		if !ok {
			updatedBy = adminID
		}
		recipe.UpdatedBy = updatedBy
		sb.Reset()
		_, _ = fmt.Fprintf(&sb, "%s/resep/%s", os.Getenv("PHOTO_DIRECTORY"), recipe.Photo)
		src = sb.String()
		sb.Reset()
		_, _ = fmt.Fprintf(&sb, "%s.webp", uuid.NewV7().String())
		recipe.Photo = sb.String()
		sb.Reset()
		_, _ = fmt.Fprintf(&sb, "%s/%s", os.Getenv("PHOTO_DIRECTORY"), recipe.Photo)
		dst = sb.String()
		photos = append(photos, dst)
		switch filepath.Ext(src) {
		case ".jpg", ".jpeg":
			if err = jpeg2webp(src, dst); err != nil {
				Log(slog.LevelError, err.Error())
				return
			}
		case ".png":
			if err = png2webp(src, dst); err != nil {
				Log(slog.LevelError, err.Error())
				return
			}
		case ".webp":
			if err = copyFile(src, dst); err != nil {
				Log(slog.LevelError, err.Error())
				return
			}
		}
		if err = tx.QueryRow(
			ctx,
			`INSERT INTO recipes (
				record_status
				, name
				, photo
				, description
				, instagram_link
				, facebook_link
				, created_by
				, created_at
				, updated_by
				, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			RETURNING id`,
			RecordStatusActive,
			recipe.Name,
			recipe.Photo,
			recipe.Description,
			recipe.InstagramLink,
			recipe.FacebookLink,
			recipe.CreatedBy,
			recipe.CreatedAt,
			recipe.UpdatedBy,
			recipe.UpdatedAt,
		).Scan(&recipe.ID); err != nil {
			Log(slog.LevelError, err.Error())
			return
		}
		for _, recipeGuide := range recipe.Guides {
			if _, err = tx.Exec(
				ctx,
				`INSERT INTO recipe_guides (
					record_status
					, recipe_id
					, instruction
					, created_by
					, created_at
					, updated_by
					, updated_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
				RecordStatusActive,
				recipe.ID,
				recipeGuide.Instruction,
				recipe.CreatedBy,
				recipe.CreatedAt,
				recipe.UpdatedBy,
				recipe.UpdatedAt,
			); err != nil {
				Log(slog.LevelError, err.Error())
				return
			}
		}
		for _, recipeItem := range recipe.Items {
			product, ok := mapProducts[recipeItem.OldProductID]
			if !ok {
				continue
			}
			if _, err = tx.Exec(
				ctx,
				`INSERT INTO recipe_items (
					record_status
					, recipe_id
					, product_id
					, product_name
					, product_photo
					, product_ready_stock
					, dose
					, note
					, created_by
					, created_at
					, updated_by
					, updated_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
				RecordStatusActive,
				recipe.ID,
				product.ID,
				product.Name,
				product.Photo,
				product.ReadyStock,
				recipeItem.Dose,
				recipeItem.Note,
				recipe.CreatedBy,
				recipe.CreatedAt,
				recipe.UpdatedBy,
				recipe.UpdatedAt,
			); err != nil {
				Log(slog.LevelError, err.Error())
				return
			}
		}
		recipes[i] = recipe
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
		for i := range photos {
			_ = os.RemoveAll(photos[i])
		}
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
	defer func() {
		_ = source.Close()
	}()
	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		_ = destination.Close()
	}()
	if _, err = io.Copy(destination, source); err != nil {
		return err
	}
	if err = destination.Sync(); err != nil {
		return err
	}
	return os.Chmod(dst, 0644)
}

func jpeg2webp(src, dst string) error {
	input, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		_ = input.Close()
	}()
	image, err := jpeg.Decode(input)
	if err != nil {
		return err
	}
	output, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		_ = output.Close()
	}()
	options, err := encoder.NewLossyEncoderOptions(encoder.PresetDefault, 80)
	if err != nil {
		return err
	}
	if err = webp.Encode(output, image, options); err != nil {
		return err
	}
	return os.Chmod(dst, 0644)
}

func png2webp(src, dst string) error {
	input, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		_ = input.Close()
	}()
	image, err := png.Decode(input)
	if err != nil {
		return err
	}
	output, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		_ = output.Close()
	}()
	options, err := encoder.NewLossyEncoderOptions(encoder.PresetDefault, 80)
	if err != nil {
		return err
	}
	if err = webp.Encode(output, image, options); err != nil {
		return err
	}
	return os.Chmod(dst, 0644)
}
