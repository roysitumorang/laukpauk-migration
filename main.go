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
	"strconv"
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
		Subject,
		Body string
		Photo,
		LinkText,
		LinkURL *string
	}

	LandingPageBlockItemData struct {
		Subject string  `json:"title"`
		Body    string  `json:"content"`
		Photo   *string `json:"img"`
	}

	LandingPageBlockItemHero struct {
		Subject  string  `json:"heading"`
		Body     string  `json:"content"`
		LinkURL  *string `json:"buttonLink"`
		LinkText *string `json:"buttonText"`
		Photo    *string `json:"img"`
	}

	Bazaar struct {
		ID    uuid.UUID
		OldID int64
		Name,
		Address,
		Photo string
		Longitude,
		Latitude float64
		BazaarType string
		OldCreatedBy,
		OldUpdatedBy int64
		CreatedBy,
		UpdatedBy *uuid.UUID
		CreatedAt,
		UpdatedAt time.Time
	}

	Seller struct {
		ID    uuid.UUID
		OldID int64
		Password,
		Username,
		Name,
		Address,
		Banner,
		Bio string
		OldBazaarID int64
		BazaarID    *uuid.UUID
		OpenOnSunday,
		OpenOnMonday,
		OpenOnTuesday,
		OpenOnWednesday,
		OpenOnThursday,
		OpenOnFriday,
		OpenOnSaturday bool
		Status           string
		StatusReviewedAt *time.Time
		OldCommissionRate,
		OldCommissionType string
		CommissionRate int64
		CommissionType,
		BankAccount string
		OldCreatedBy,
		OldUpdatedBy int64
		CreatedBy,
		UpdatedBy *uuid.UUID
		CreatedAt,
		UpdatedAt time.Time
	}

	ProductCategory struct {
		ID    uuid.UUID
		OldID int64
		Name,
		Photo string
		OldCreatedBy,
		OldUpdatedBy int64
		CreatedBy,
		UpdatedBy *uuid.UUID
		CreatedAt,
		UpdatedAt time.Time
	}

	Product struct {
		ID uuid.UUID
		OldID,
		OldSellerID,
		OldProductCategoryID int64
		SellerID          uuid.UUID
		ProductCategoryID *uuid.UUID
		Name,
		Photo,
		ShortDescription,
		FullDescription,
		StockStatus string
		Pinned bool
		OldCreatedBy,
		OldUpdatedBy int64
		CreatedBy,
		UpdatedBy *uuid.UUID
		CreatedAt,
		UpdatedAt time.Time
	}

	ProductVariant struct {
		ID uuid.UUID
		OldID,
		OldProductID int64
		ProductID         *uuid.UUID
		UnitOfMeasurement string
		Price,
		Discount int64
		PriceType              string
		PriceVariantPercentage float64
		Description            string
		OldCreatedBy,
		OldUpdatedBy int64
		CreatedBy,
		UpdatedBy *uuid.UUID
		CreatedAt,
		UpdatedAt time.Time
	}

	Voucher struct {
		ID    uuid.UUID
		OldID int64
		Code,
		Photo,
		Name string
		Amount int64
		VoucherType,
		TargetUser,
		Description string
		MinimumTransaction,
		MaximumUsage int64
		EffectiveDateStart,
		EffectiveDateEnd time.Time
		OldCreatedBy,
		OldUpdatedBy int64
		CreatedBy,
		UpdatedBy *uuid.UUID
		CreatedAt,
		UpdatedAt time.Time
	}

	VoucherUser struct {
		OldVoucherID,
		OldUserID int64
		VoucherID,
		UserID uuid.UUID
		CreatedAt time.Time
	}

	PaymentMethod struct {
		ID            uuid.UUID
		RecordStatus  int
		ChannelCode   string
		OrdinalNumber int64
		BusinessID,
		Name,
		ChannelType string
		MinimumTransaction int64
		OldCreatedBy,
		OldUpdatedBy int64
		CreatedBy,
		UpdatedBy *uuid.UUID
		CreatedAt,
		UpdatedAt time.Time
		DeletedAt *time.Time
	}

	ShippingPlan struct {
		ID           uuid.UUID
		OldID        int64
		RecordStatus int
		Name         string
		TransactionTimeStart,
		TransactionTimeEnd,
		TransactionTimeDescription,
		ShippingTimeStart,
		ShippingTimeEnd,
		ShippingTimeDescription,
		Period string
		DailyQuota,
		OrdinalNumber int64
		ShippingType string
		OldCreatedBy,
		OldUpdatedBy int64
		CreatedBy,
		UpdatedBy *uuid.UUID
		CreatedAt,
		UpdatedAt time.Time
		DeletedAt *time.Time
	}

	ShippingPlanSeller struct {
		OldShippingPlanID,
		OldSellerID int64
		ShippingPlanID,
		SellerID uuid.UUID
		CreatedAt time.Time
	}

	Transaction struct {
		ID            uuid.UUID
		OldID         int64
		BuyerID       uuid.UUID
		OldBuyerID    int64
		TransactionNo string
		DownPayment   int64
		VoucherID     *uuid.UUID
		OldVoucherID  int64
		VoucherType   *string
		VoucherAmount *int64
		RecipientName,
		RecipientMobilePhone,
		BuyerNote string
		PaymentMethodID uuid.UUID
		PaymentMethodName,
		PaymentMethodChannelCode string
		PaymentMethodExtraData *string
		PaymentNo              *string
		PaymentStatus,
		ShippingStatus string
		ReviewedBy       *uuid.UUID
		OldReviewedBy    *int64
		ReviewedAt       *time.Time
		RejectionReason  *string
		ShippingCost     int64
		ShippingCostType string
		WalletDeduction  int64
		Cashback         int64
		ShippingLongitude,
		ShippingLatitude float64
		ShippingAddress     string
		ShippingInstruction string
		ShippingDate        time.Time
		ShippingPlanID      uuid.UUID
		OldShippingPlanID   int64
		ShippingType,
		ShippingTimeline string
		RefundedShippingCost,
		RefundedTransactionCost,
		UniqueCode,
		MinimumPoint,
		PointDeduction int64
		ProductCategoryDiscountAmount int64
		PromoDiscount                 *int64
		CreatedAt                     time.Time
	}

	TransactionItem struct {
		ID                  uuid.UUID
		OldID               int64
		TransactionID       *uuid.UUID
		OldTransactionID    int64
		BazaarID            uuid.UUID
		OldBazaarID         int64
		SellerID            uuid.UUID
		OldSellerID         int64
		ProductVariantID    *uuid.UUID
		OldProductVariantID int64
		ProductID           *uuid.UUID
		OldProductID        int64
		ProductType         string
		PromoID             *uuid.UUID
		OldPromoID          int64
		PromoDiscount,
		Price,
		Discount,
		Quantity int64
		Status          string
		ReviewedBy      *uuid.UUID
		OldReviewedBy   int64
		ReviewedAt      *time.Time
		RejectionReason *string
		BuyerNote       string
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
				`INSERT INTO landing_page_block_items (
					record_status
					, landing_page_block_name
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
	// bazaars
	rows, err = mysql.QueryContext(
		ctx,
		`SELECT
			id
			, nama
			, alamat
			, gambar
			, latitude
			, longitude
			, CASE pasarUmum WHEN 1 THEN 'public' ELSE 'private' END
			, createdBy
			, createdAt
			, updatedBy
			, updatedAt
		FROM pasar
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
	var bazaars []Bazaar
	for rows.Next() {
		var bazaar Bazaar
		if err = rows.Scan(
			&bazaar.OldID,
			&bazaar.Name,
			&bazaar.Address,
			&bazaar.Photo,
			&bazaar.Latitude,
			&bazaar.Longitude,
			&bazaar.BazaarType,
			&bazaar.OldCreatedBy,
			&bazaar.CreatedAt,
			&bazaar.OldUpdatedBy,
			&bazaar.UpdatedAt,
		); err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
			Log(slog.LevelError, err.Error())
			return
		}
		bazaars = append(bazaars, bazaar)
	}
	mapBazaarIDs := map[int64]uuid.UUID{}
	for i, bazaar := range bazaars {
		if createdBy, ok := mapAdminIDs[bazaar.OldCreatedBy]; ok {
			bazaar.CreatedBy = &createdBy
		}
		if updatedBy, ok := mapAdminIDs[bazaar.OldUpdatedBy]; ok {
			bazaar.UpdatedBy = &updatedBy
		}
		if err = tx.QueryRow(
			ctx,
			`INSERT INTO bazaars (
				record_status
				, name
				, address
				, photo
				, coordinates
				, bazaar_type
				, created_by
				, created_at
				, updated_by
				, updated_at
			) VALUES ($1, $2, $3, $4, POINT($5, $6), $7, $8, $9, $10, $11)
			RETURNING id`,
			1,
			bazaar.Name,
			bazaar.Address,
			bazaar.Photo,
			bazaar.Longitude,
			bazaar.Latitude,
			bazaar.BazaarType,
			bazaar.CreatedBy,
			bazaar.CreatedAt,
			bazaar.UpdatedBy,
			bazaar.UpdatedAt,
		).Scan(&bazaar.ID); err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
			Log(slog.LevelError, err.Error())
			return
		}
		bazaars[i] = bazaar
		mapBazaarIDs[bazaar.OldID] = bazaar.ID
	}
	// sellers
	rows, err = mysql.QueryContext(
		ctx,
		`SELECT
			id
			, namaToko
			, alamatToko
			, banner
			, bio
			, pasar
			, openOnSunday = 1
			, openOnMonday = 1
			, openOnTuesday = 1
			, openOnWednesday = 1
			, openOnThursday = 1
			, openOnFriday = 1
			, openOnSaturday = 1
			, status
			, CASE WHEN status = 'pending' THEN NULL ELSE statusDoneAt END
			, nilaiKerjasama
			, satuanNilaiKerjasama
			, nomorRekening
			, createdBy
			, createdAt
			, updatedBy
			, updatedAt
		FROM penjual
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
	var sellers []Seller
	for rows.Next() {
		var seller Seller
		if err = rows.Scan(
			&seller.OldID,
			&seller.Name,
			&seller.Address,
			&seller.Banner,
			&seller.Bio,
			&seller.OldBazaarID,
			&seller.OpenOnSunday,
			&seller.OpenOnMonday,
			&seller.OpenOnTuesday,
			&seller.OpenOnWednesday,
			&seller.OpenOnThursday,
			&seller.OpenOnFriday,
			&seller.OpenOnSaturday,
			&seller.Status,
			&seller.StatusReviewedAt,
			&seller.OldCommissionRate,
			&seller.OldCommissionType,
			&seller.BankAccount,
			&seller.OldCreatedBy,
			&seller.CreatedAt,
			&seller.OldUpdatedBy,
			&seller.UpdatedAt,
		); err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
			Log(slog.LevelError, err.Error())
			return
		}
		if bazaarID, ok := mapBazaarIDs[seller.OldBazaarID]; ok {
			seller.BazaarID = &bazaarID
		}
		if createdBy, ok := mapAdminIDs[seller.OldCreatedBy]; ok {
			seller.CreatedBy = &createdBy
		}
		if updatedBy, ok := mapAdminIDs[seller.OldUpdatedBy]; ok {
			seller.UpdatedBy = &updatedBy
		}
		hashByte, err := bcrypt.GenerateFromPassword(temporaryPassword, bcrypt.MinCost)
		if err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
			Log(slog.LevelError, err.Error())
			return
		}
		seller.Password = ByteSlice2String(hashByte)
		seller.Username = RandomString(6)
		for {
			if _, ok := mapUsernames[seller.Username]; !ok {
				mapUsernames[seller.Username] = struct{}{}
				break
			}
			seller.Username = RandomString(6)
		}
		if commissionRate, _ := strconv.ParseInt(seller.OldCommissionRate, 10, 64); commissionRate > 0 {
			seller.CommissionRate = commissionRate
		}
		switch seller.OldCommissionType {
		case "persentase":
			seller.CommissionType = "percentage"
		case "rupiah":
			seller.CommissionType = "flat"
		}
		sellers = append(sellers, seller)
	}
	mapSellerIDs := map[int64]uuid.UUID{}
	for i, seller := range sellers {
		if err = tx.QueryRow(
			ctx,
			`INSERT INTO users (
				user_type
				, name
				, username
				, encrypted_password
				, created_by
				, created_at
				, updated_by
				, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id`,
			"seller",
			seller.Name,
			seller.Username,
			seller.Password,
			seller.CreatedBy,
			seller.CreatedAt,
			seller.UpdatedBy,
			seller.UpdatedAt,
		).Scan(&seller.ID); err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
			Log(slog.LevelError, err.Error())
			return
		}
		if _, err = tx.Exec(
			ctx,
			`INSERT INTO sellers (
				user_id
				, address
				, banner
				, bio
				, bazaar_id
				, open_on_sunday
				, open_on_monday
				, open_on_tuesday
				, open_on_wednesday
				, open_on_thursday
				, open_on_friday
				, open_on_saturday
				, status
				, status_reviewed_at
				, commission_rate
				, commission_type
				, bank_account
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9
				, $10, $11, $12, $13, $14, $15, $16, $17
			)`,
			&seller.ID,
			&seller.Address,
			&seller.Banner,
			&seller.Bio,
			&seller.BazaarID,
			&seller.OpenOnSunday,
			&seller.OpenOnMonday,
			&seller.OpenOnTuesday,
			&seller.OpenOnWednesday,
			&seller.OpenOnThursday,
			&seller.OpenOnFriday,
			&seller.OpenOnSaturday,
			&seller.Status,
			&seller.StatusReviewedAt,
			&seller.CommissionRate,
			&seller.CommissionType,
			&seller.BankAccount,
		); err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
			Log(slog.LevelError, err.Error())
			return
		}
		sellers[i] = seller
		mapSellerIDs[seller.OldID] = seller.ID
	}
	// product_categories
	rows, err = mysql.QueryContext(
		ctx,
		`SELECT
			id
			, nama
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
		if errRollback := tx.Rollback(ctx); errRollback != nil {
			Log(slog.LevelError, errRollback.Error())
		}
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
			&productCategory.Photo,
			&productCategory.OldCreatedBy,
			&productCategory.CreatedAt,
			&productCategory.OldUpdatedBy,
			&productCategory.UpdatedAt,
		); err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
			Log(slog.LevelError, err.Error())
			return
		}
		if createdBy, ok := mapAdminIDs[productCategory.OldCreatedBy]; ok {
			productCategory.CreatedBy = &createdBy
		}
		if updatedBy, ok := mapAdminIDs[productCategory.OldUpdatedBy]; ok {
			productCategory.UpdatedBy = &updatedBy
		}
		productCategories = append(productCategories, productCategory)
	}
	mapProductCategoryIDs := map[int64]uuid.UUID{}
	for i, productCategory := range productCategories {
		if err = tx.QueryRow(
			ctx,
			`INSERT INTO product_categories (
				record_status
				, name
				, photo
				, created_by
				, created_at
				, updated_by
				, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id`,
			1,
			productCategory.Name,
			productCategory.Photo,
			productCategory.CreatedBy,
			productCategory.CreatedAt,
			productCategory.UpdatedBy,
			productCategory.UpdatedAt,
		).Scan(&productCategory.ID); err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
			Log(slog.LevelError, err.Error())
			return
		}
		productCategories[i] = productCategory
		mapProductCategoryIDs[productCategory.OldID] = productCategory.ID
	}
	// standalone products
	rows, err = mysql.QueryContext(
		ctx,
		`SELECT
			id
			, idPenjual
			, foto
			, nama
			, idKategoriProduk
			, COALESCE(keteranganSingkat, '')
			, keterangan
			, CASE WHEN stokAvailable = 1 THEN 'available' ELSE 'unavailable' END
			, superior = 1
			, createdBy
			, createdAt
			, updatedBy
			, updatedAt
		FROM produk
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
	var products []Product
	for rows.Next() {
		var product Product
		if err = rows.Scan(
			&product.OldID,
			&product.OldSellerID,
			&product.Photo,
			&product.Name,
			&product.OldProductCategoryID,
			&product.ShortDescription,
			&product.FullDescription,
			&product.StockStatus,
			&product.Pinned,
			&product.OldCreatedBy,
			&product.CreatedAt,
			&product.OldUpdatedBy,
			&product.UpdatedAt,
		); err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
			Log(slog.LevelError, err.Error())
			return
		}
		if sellerID, ok := mapSellerIDs[product.OldSellerID]; ok {
			product.SellerID = sellerID
		}
		if productCategoryID, ok := mapProductCategoryIDs[product.OldProductCategoryID]; ok {
			product.ProductCategoryID = &productCategoryID
		}
		if createdBy, ok := mapAdminIDs[product.OldCreatedBy]; ok {
			product.CreatedBy = &createdBy
		}
		if updatedBy, ok := mapAdminIDs[product.OldUpdatedBy]; ok {
			product.UpdatedBy = &updatedBy
		}
		products = append(products, product)
	}
	mapProductIDs := map[int64]uuid.UUID{}
	for i, product := range products {
		if err = tx.QueryRow(
			ctx,
			`INSERT INTO products (
				record_status
				, seller_id
				, product_category_id
				, product_type
				, name
				, photo
				, short_description
				, full_description
				, stock_status
				, pinned
				, created_by
				, created_at
				, updated_by
				, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
			RETURNING id`,
			1,
			product.SellerID,
			product.ProductCategoryID,
			"non bundled",
			product.Name,
			product.Photo,
			product.ShortDescription,
			product.FullDescription,
			product.StockStatus,
			product.Pinned,
			product.CreatedBy,
			product.CreatedAt,
			product.UpdatedBy,
			product.UpdatedAt,
		).Scan(&product.ID); err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
			Log(slog.LevelError, err.Error())
			return
		}
		products[i] = product
		mapProductIDs[product.OldID] = product.ID
	}
	// standalone product_variants
	rows, err = mysql.QueryContext(
		ctx,
		`SELECT
			h.id
			, h.idProduk
			, h.nama
			, h.harga
			, h.diskon
			, CASE WHEN h.jenisHarga = 'pas' THEN 'exact' ELSE 'inexact' END
			, COALESCE(h.persentaseHargaVariasi, 0)
			, h.keterangan
			, p.createdBy
			, p.createdAt
			, p.updatedBy
			, p.updatedAt
		FROM produk_harga h
		JOIN produk p ON h.idProduk = p.id
		ORDER BY h.id`,
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
	var productVariants []ProductVariant
	for rows.Next() {
		var productVariant ProductVariant
		if err = rows.Scan(
			&productVariant.OldID,
			&productVariant.OldProductID,
			&productVariant.UnitOfMeasurement,
			&productVariant.Price,
			&productVariant.Discount,
			&productVariant.PriceType,
			&productVariant.PriceVariantPercentage,
			&productVariant.Description,
			&productVariant.OldCreatedBy,
			&productVariant.CreatedAt,
			&productVariant.OldUpdatedBy,
			&productVariant.UpdatedAt,
		); err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
			Log(slog.LevelError, err.Error())
			return
		}
		if productID, ok := mapProductIDs[productVariant.OldProductID]; ok {
			productVariant.ProductID = &productID
		}
		if createdBy, ok := mapAdminIDs[productVariant.OldCreatedBy]; ok {
			productVariant.CreatedBy = &createdBy
		}
		if updatedBy, ok := mapAdminIDs[productVariant.OldUpdatedBy]; ok {
			productVariant.UpdatedBy = &updatedBy
		}
		productVariants = append(productVariants, productVariant)
	}
	mapProductVariantIDs := map[int64]uuid.UUID{}
	mapVariantKeys := map[string]struct{}{}
	for i, productVariant := range productVariants {
		sb.Reset()
		_, _ = fmt.Fprintf(&sb, "%d%s", productVariant.OldProductID, productVariant.UnitOfMeasurement)
		variantKey := sb.String()
		if _, ok := mapVariantKeys[variantKey]; ok {
			continue
		}
		if err = tx.QueryRow(
			ctx,
			`INSERT INTO product_variants (
				record_status
				, product_id
				, unit_of_measurement
				, price
				, discount
				, price_type
				, price_variant_percentage
				, description
				, created_by
				, created_at
				, updated_by
				, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			RETURNING id`,
			1,
			productVariant.ProductID,
			productVariant.UnitOfMeasurement,
			productVariant.Price,
			productVariant.Discount,
			productVariant.PriceType,
			productVariant.PriceVariantPercentage,
			productVariant.Description,
			productVariant.CreatedBy,
			productVariant.CreatedAt,
			productVariant.UpdatedBy,
			productVariant.UpdatedAt,
		).Scan(&productVariant.ID); err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
			Log(slog.LevelError, err.Error())
			return
		}
		productVariants[i] = productVariant
		mapProductVariantIDs[productVariant.OldID] = productVariant.ID
		mapVariantKeys[variantKey] = struct{}{}
	}
	// bundled products
	rows, err = mysql.QueryContext(
		ctx,
		`SELECT
			id
			, idPenjual
			, foto
			, nama
			, harga
			, diskon
			, CASE WHEN stockAvailable = 1 THEN 'available' ELSE 'unavailable' END
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
		if errRollback := tx.Rollback(ctx); errRollback != nil {
			Log(slog.LevelError, errRollback.Error())
		}
		Log(slog.LevelError, err.Error())
		return
	}
	defer rows.Close()
	products = []Product{}
	productVariants = []ProductVariant{}
	for rows.Next() {
		var (
			product        Product
			productVariant ProductVariant
		)
		if err = rows.Scan(
			&product.OldID,
			&product.OldSellerID,
			&product.Photo,
			&product.Name,
			&productVariant.Price,
			&productVariant.Discount,
			&product.StockStatus,
			&product.Pinned,
			&product.FullDescription,
			&product.OldCreatedBy,
			&product.CreatedAt,
			&product.OldUpdatedBy,
			&product.UpdatedAt,
		); err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
			Log(slog.LevelError, err.Error())
			return
		}
		productVariant.OldProductID = product.OldID
		if sellerID, ok := mapSellerIDs[product.OldSellerID]; ok {
			product.SellerID = sellerID
		}
		if createdBy, ok := mapAdminIDs[product.OldCreatedBy]; ok {
			product.CreatedBy = &createdBy
			productVariant.CreatedBy = &createdBy
		}
		if updatedBy, ok := mapAdminIDs[product.OldUpdatedBy]; ok {
			product.UpdatedBy = &updatedBy
			productVariant.UpdatedBy = &updatedBy
		}
		products = append(products, product)
		productVariants = append(productVariants, productVariant)
	}
	for i, product := range products {
		if err = tx.QueryRow(
			ctx,
			`INSERT INTO products (
				record_status
				, seller_id
				, product_category_id
				, product_type
				, name
				, photo
				, short_description
				, full_description
				, stock_status
				, pinned
				, created_by
				, created_at
				, updated_by
				, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
			RETURNING id`,
			1,
			product.SellerID,
			product.ProductCategoryID,
			"bundled",
			product.Name,
			product.Photo,
			product.ShortDescription,
			product.FullDescription,
			product.StockStatus,
			product.Pinned,
			product.CreatedBy,
			product.CreatedAt,
			product.UpdatedBy,
			product.UpdatedAt,
		).Scan(&product.ID); err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
			Log(slog.LevelError, err.Error())
			return
		}
		products[i] = product
		mapProductIDs[product.OldID] = product.ID
	}
	for i, productVariant := range productVariants {
		if productID, ok := mapProductIDs[productVariant.OldProductID]; ok {
			productVariant.ProductID = &productID
		}
		if err = tx.QueryRow(
			ctx,
			`INSERT INTO product_variants (
				record_status
				, product_id
				, unit_of_measurement
				, price
				, discount
				, price_type
				, price_variant_percentage
				, description
				, created_by
				, created_at
				, updated_by
				, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			RETURNING id`,
			1,
			productVariant.ProductID,
			"paket",
			productVariant.Price,
			productVariant.Discount,
			productVariant.PriceType,
			productVariant.PriceVariantPercentage,
			productVariant.Description,
			productVariant.CreatedBy,
			productVariant.CreatedAt,
			productVariant.UpdatedBy,
			productVariant.UpdatedAt,
		).Scan(&productVariant.ID); err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
			Log(slog.LevelError, err.Error())
			return
		}
		productVariants[i] = productVariant
		mapProductVariantIDs[productVariant.OldID] = productVariant.ID
	}
	// vouchers
	rows, err = mysql.QueryContext(
		ctx,
		`SELECT
			id
			, kode
			, foto
			, nama
			, nilai
			, CASE WHEN jenis = 'belanja' THEN 'shopping cost' ELSE 'shipping cost' END
			, CASE WHEN isGlobal = 1 THEN 'all' ELSE 'specific' END
			, deskripsi
			, minimumBelanja
			, maximumUsage
			, berlakuMulai
			, berlakuSampai
			, createdBy
			, createdAt
			, updatedBy
			, updatedAt
		FROM voucher
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
	var vouchers []Voucher
	for rows.Next() {
		var voucher Voucher
		if err = rows.Scan(
			&voucher.OldID,
			&voucher.Code,
			&voucher.Photo,
			&voucher.Name,
			&voucher.Amount,
			&voucher.VoucherType,
			&voucher.TargetUser,
			&voucher.Description,
			&voucher.MinimumTransaction,
			&voucher.MaximumUsage,
			&voucher.EffectiveDateStart,
			&voucher.EffectiveDateEnd,
			&voucher.OldCreatedBy,
			&voucher.CreatedAt,
			&voucher.OldUpdatedBy,
			&voucher.UpdatedAt,
		); err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
			Log(slog.LevelError, err.Error())
			return
		}
		if createdBy, ok := mapAdminIDs[voucher.OldCreatedBy]; ok {
			voucher.CreatedBy = &createdBy
		}
		if updatedBy, ok := mapAdminIDs[voucher.OldUpdatedBy]; ok {
			voucher.UpdatedBy = &updatedBy
		}
		vouchers = append(vouchers, voucher)
	}
	mapVoucherIDs := map[int64]uuid.UUID{}
	duration := (24 * time.Hour) - time.Nanosecond
	for i, voucher := range vouchers {
		if err = tx.QueryRow(
			ctx,
			`INSERT INTO vouchers (
				record_status
				, code
				, photo
				, name
				, amount
				, voucher_type
				, target_user
				, description
				, minimum_transaction
				, maximum_usage
				, effective_date_start
				, effective_date_end
				, created_by
				, created_at
				, updated_by
				, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
			RETURNING id`,
			1,
			voucher.Code,
			voucher.Photo,
			voucher.Name,
			voucher.Amount,
			voucher.VoucherType,
			voucher.TargetUser,
			voucher.Description,
			voucher.MinimumTransaction,
			voucher.MaximumUsage,
			voucher.EffectiveDateStart,
			voucher.EffectiveDateEnd.Add(duration),
			voucher.CreatedBy,
			voucher.CreatedAt,
			voucher.UpdatedBy,
			voucher.UpdatedAt,
		).Scan(&voucher.ID); err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
			Log(slog.LevelError, err.Error())
			return
		}
		vouchers[i] = voucher
		mapVoucherIDs[voucher.OldID] = voucher.ID
	}
	// vouchers_users
	rows, err = mysql.QueryContext(
		ctx,
		`SELECT
			vu.idVoucher
			, vu.idUser
			, v.createdAt
		FROM voucher_user vu
		JOIN voucher v ON vu.idVoucher = v.id
		ORDER BY vu.id`,
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
	var vouchersUsers []VoucherUser
	for rows.Next() {
		var voucherUser VoucherUser
		if err = rows.Scan(
			&voucherUser.OldVoucherID,
			&voucherUser.OldUserID,
			&voucherUser.CreatedAt,
		); err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
			Log(slog.LevelError, err.Error())
			return
		}
		voucherID, voucherIdOk := mapVoucherIDs[voucherUser.OldVoucherID]
		userID, userIdOk := mapBuyerIDs[voucherUser.OldUserID]
		if voucherIdOk && userIdOk {
			voucherUser.VoucherID = voucherID
			voucherUser.UserID = userID
			vouchersUsers = append(vouchersUsers, voucherUser)
		}
	}
	for i, voucherUser := range vouchersUsers {
		if _, err = tx.Exec(
			ctx,
			`INSERT INTO vouchers_users (
				voucher_id
				, user_id
				, created_at
			) VALUES ($1, $2, $3)`,
			voucherUser.VoucherID,
			voucherUser.UserID,
			voucherUser.CreatedAt,
		); err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
			Log(slog.LevelError, err.Error())
			return
		}
		vouchersUsers[i] = voucherUser
	}
	// payment_methods
	rows, err = mysql.QueryContext(
		ctx,
		`SELECT
			channel_code
			, urutan
			, business_id
			, name
			, channel_category
			, CASE WHEN is_enabled = 1 THEN 1 ELSE -1 END
			, minimalPembayaran
		FROM metode_pembayaran
		ORDER BY urutan`,
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
	var paymentMethods []PaymentMethod
	for rows.Next() {
		var paymentMethod PaymentMethod
		if err = rows.Scan(
			&paymentMethod.ChannelCode,
			&paymentMethod.OrdinalNumber,
			&paymentMethod.BusinessID,
			&paymentMethod.Name,
			&paymentMethod.ChannelType,
			&paymentMethod.RecordStatus,
			&paymentMethod.MinimumTransaction,
		); err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
			Log(slog.LevelError, err.Error())
			return
		}
		paymentMethod.CreatedAt = now
		paymentMethod.UpdatedAt = now
		if paymentMethod.RecordStatus == -1 {
			paymentMethod.DeletedAt = &now
		}
		paymentMethods = append(paymentMethods, paymentMethod)
	}
	mapPaymentMethodIDs := map[string]uuid.UUID{}
	for i, paymentMethod := range paymentMethods {
		if err = tx.QueryRow(
			ctx,
			`INSERT INTO payment_methods (
				record_status
				, channel_code
				, ordinal_number
				, business_id
				, name
				, channel_type
				, minimum_transaction
				, created_at
				, updated_at
				, deleted_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			RETURNING id`,
			paymentMethod.RecordStatus,
			paymentMethod.ChannelCode,
			paymentMethod.OrdinalNumber,
			paymentMethod.BusinessID,
			paymentMethod.Name,
			paymentMethod.ChannelType,
			paymentMethod.MinimumTransaction,
			paymentMethod.CreatedAt,
			paymentMethod.UpdatedAt,
			paymentMethod.DeletedAt,
		).Scan(&paymentMethod.ID); err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
			Log(slog.LevelError, err.Error())
			return
		}
		paymentMethods[i] = paymentMethod
		mapPaymentMethodIDs[paymentMethod.ChannelCode] = paymentMethod.ID
	}
	// shipping_plans
	rows, err = mysql.QueryContext(
		ctx,
		`SELECT
			id
			, CASE WHEN active = 1 THEN 1 ELSE -1 END
			, minTime
			, maxTime
			, readAbleMinMaxTime
			, minJadwalDelivery
			, maxJadwalDelivery
			, readAbleMinMaxJadwalDelivery
			, sessi
			, quota
			, urutan
			, CASE WHEN sameDay = 1 THEN 'same day' ELSE 'next day' END
			, keterangan
		FROM jadwal_delivery
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
	var shippingPlans []ShippingPlan
	for rows.Next() {
		var shippingPlan ShippingPlan
		if err = rows.Scan(
			&shippingPlan.OldID,
			&shippingPlan.RecordStatus,
			&shippingPlan.TransactionTimeStart,
			&shippingPlan.TransactionTimeEnd,
			&shippingPlan.TransactionTimeDescription,
			&shippingPlan.ShippingTimeStart,
			&shippingPlan.ShippingTimeEnd,
			&shippingPlan.ShippingTimeDescription,
			&shippingPlan.Period,
			&shippingPlan.DailyQuota,
			&shippingPlan.OrdinalNumber,
			&shippingPlan.ShippingType,
			&shippingPlan.Name,
		); err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
			Log(slog.LevelError, err.Error())
			return
		}
		shippingPlan.CreatedAt = now
		shippingPlan.UpdatedAt = now
		if shippingPlan.RecordStatus == -1 {
			shippingPlan.DeletedAt = &now
		}
		shippingPlans = append(shippingPlans, shippingPlan)
	}
	mapShippingPlanIDs := map[int64]uuid.UUID{}
	for i, shippingPlan := range shippingPlans {
		if err = tx.QueryRow(
			ctx,
			`INSERT INTO shipping_plans (
				record_status
				, name
				, transaction_time_start
				, transaction_time_end
				, transaction_time_description
				, shipping_time_start
				, shipping_time_end
				, shipping_time_description
				, period
				, daily_quota
				, ordinal_number
				, shipping_type
				, created_at
				, updated_at
				, deleted_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
			RETURNING id`,
			shippingPlan.RecordStatus,
			shippingPlan.Name,
			shippingPlan.TransactionTimeStart,
			shippingPlan.TransactionTimeEnd,
			shippingPlan.TransactionTimeDescription,
			shippingPlan.ShippingTimeStart,
			shippingPlan.ShippingTimeEnd,
			shippingPlan.ShippingTimeDescription,
			shippingPlan.Period,
			shippingPlan.DailyQuota,
			shippingPlan.OrdinalNumber,
			shippingPlan.ShippingType,
			shippingPlan.CreatedAt,
			shippingPlan.UpdatedAt,
			shippingPlan.DeletedAt,
		).Scan(&shippingPlan.ID); err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
			Log(slog.LevelError, err.Error())
			return
		}
		shippingPlans[i] = shippingPlan
		mapShippingPlanIDs[shippingPlan.OldID] = shippingPlan.ID
	}
	// shipping_plans_sellers
	rows, err = mysql.QueryContext(
		ctx,
		`SELECT
			jadwalDeliveryCode
			, idPenjual
		FROM jadwal_delivery_penjual
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
	var shippingPlansSellers []ShippingPlanSeller
	for rows.Next() {
		var shippingPlanSeller ShippingPlanSeller
		if err = rows.Scan(
			&shippingPlanSeller.OldShippingPlanID,
			&shippingPlanSeller.OldSellerID,
		); err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
			Log(slog.LevelError, err.Error())
			return
		}
		shippingPlanSeller.CreatedAt = now
		shippingPlanID, shippingPlanIdOk := mapShippingPlanIDs[shippingPlanSeller.OldShippingPlanID]
		sellerID, sellerIdOk := mapSellerIDs[shippingPlanSeller.OldSellerID]
		if shippingPlanIdOk && sellerIdOk {
			shippingPlanSeller.ShippingPlanID = shippingPlanID
			shippingPlanSeller.SellerID = sellerID
			shippingPlansSellers = append(shippingPlansSellers, shippingPlanSeller)
		}
	}
	for _, shippingPlanSeller := range shippingPlansSellers {
		if _, err = tx.Exec(
			ctx,
			`INSERT INTO shipping_plans_sellers (
				shipping_plan_id
				, seller_id
				, created_at
			) VALUES ($1, $2, $3)`,
			shippingPlanSeller.ShippingPlanID,
			shippingPlanSeller.SellerID,
			shippingPlanSeller.CreatedAt,
		); err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
			Log(slog.LevelError, err.Error())
			return
		}
	}
	// transactions
	rows, err = mysql.QueryContext(
		ctx,
		`SELECT
			id
			, idUser
			, nomorTransaksi
			, bayar
			, idVoucher
			, jenisVoucher
			, nilaiVoucher
			, namaPenerima
			, noHPPenerima
			, catatan
			, waktuCheckout
			, paymentChannelCode
			, namaMetodePembayaran
			, moreDataMetodePembayaran
			, nomorPembayaran
			, statusPembayaran
			, status
			, CASE WHEN statusDoneBy != 'pembeli' THEN statusDoneBy END
			, statusDoneAt
			, statusReason
			, ongkir
			, jenisOngkir
			, potonganWallet
			, cashback
			, latDelivery
			, longDelivery
			, COALESCE(alamatDelivery, '')
			, alamatDetailLengkap
			, tanggalDelivery
			, jadwalDeliveryCode
			, CASE dayDelivery
				WHEN 'Hari Ini' THEN 'same day'
				WHEN 'Besok' THEN 'next day'
				WHEN 'Lusa' THEN 'two days'
			END
			, jadwalDelivery
			, refundedOngkir
			, refundedUangBelanja
			, COALESCE(kodeUnik, 0)
			, minimalPoint
			, pointOut
			, COALESCE(diskonKategori, 0)
			, diskonPromo
		FROM transaksi
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
	var transactions []Transaction
	mapUniqueCodes := map[int64]struct{}{}
	for rows.Next() {
		var transaction Transaction
		if err = rows.Scan(
			&transaction.OldID,
			&transaction.OldBuyerID,
			&transaction.TransactionNo,
			&transaction.DownPayment,
			&transaction.OldVoucherID,
			&transaction.VoucherType,
			&transaction.VoucherAmount,
			&transaction.RecipientName,
			&transaction.RecipientMobilePhone,
			&transaction.BuyerNote,
			&transaction.CreatedAt,
			&transaction.PaymentMethodChannelCode,
			&transaction.PaymentMethodName,
			&transaction.PaymentMethodExtraData,
			&transaction.PaymentNo,
			&transaction.PaymentStatus,
			&transaction.ShippingStatus,
			&transaction.OldReviewedBy,
			&transaction.ReviewedAt,
			&transaction.RejectionReason,
			&transaction.ShippingCost,
			&transaction.ShippingCostType,
			&transaction.WalletDeduction,
			&transaction.Cashback,
			&transaction.ShippingLatitude,
			&transaction.ShippingLongitude,
			&transaction.ShippingAddress,
			&transaction.ShippingInstruction,
			&transaction.ShippingDate,
			&transaction.OldShippingPlanID,
			&transaction.ShippingType,
			&transaction.ShippingTimeline,
			&transaction.RefundedShippingCost,
			&transaction.RefundedTransactionCost,
			&transaction.UniqueCode,
			&transaction.MinimumPoint,
			&transaction.PointDeduction,
			&transaction.ProductCategoryDiscountAmount,
			&transaction.PromoDiscount,
		); err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
			Log(slog.LevelError, err.Error())
			return
		}
		buyerID, ok := mapBuyerIDs[transaction.OldBuyerID]
		if ok {
			transaction.BuyerID = buyerID
		}
		voucherID, ok := mapVoucherIDs[transaction.OldVoucherID]
		if ok {
			transaction.VoucherID = &voucherID
		}
		paymentMethodID, ok := mapPaymentMethodIDs[transaction.PaymentMethodChannelCode]
		if ok {
			transaction.PaymentMethodID = paymentMethodID
		}
		if transaction.OldReviewedBy != nil {
			reviewedBy, ok := mapShippingPlanIDs[*transaction.OldReviewedBy]
			if ok {
				transaction.ReviewedBy = &reviewedBy
			}
		}
		shippingPlanID, ok := mapShippingPlanIDs[transaction.OldShippingPlanID]
		if ok {
			transaction.ShippingPlanID = shippingPlanID
		}
		if transaction.UniqueCode == 0 {
			transaction.UniqueCode, _ = strconv.ParseInt(RandomNumber(6), 10, 64)
		}
		for {
			if _, ok := mapUniqueCodes[transaction.UniqueCode]; !ok {
				mapUniqueCodes[transaction.UniqueCode] = struct{}{}
				break
			}
			transaction.UniqueCode, _ = strconv.ParseInt(RandomNumber(6), 10, 64)
		}
		transactions = append(transactions, transaction)
	}
	mapTransactionIDs := map[int64]uuid.UUID{}
	for _, transaction := range transactions {
		if err = tx.QueryRow(
			ctx,
			`INSERT INTO transactions (
				buyer_id
				, transaction_no
				, down_payment
				, voucher_id
				, voucher_type
				, voucher_amount
				, recipient_name
				, recipient_mobile_phone
				, buyer_note
				, payment_method_id
				, payment_method_name
				, payment_method_channel_code
				, payment_method_extra_data
				, payment_no
				, payment_status
				, shipping_status
				, reviewed_by
				, reviewed_at
				, rejection_reason
				, shipping_cost
				, shipping_cost_type
				, wallet_deduction
				, cashback
				, shipping_coordinates
				, shipping_address
				, shipping_instruction
				, shipping_date
				, shipping_plan_id
				, shipping_type
				, shipping_timeline
				, refunded_shipping_cost
				, refunded_transaction_cost
				, unique_code
				, minimum_point
				, point_deduction
				, product_category_discount_amount
				, promo_discount
				, created_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9
				, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19
				, $20, $21, $22, $23, POINT($24, $25), $26, $27, $28, $29
				, $30, $31, $32, $33, $34, $35, $36, $37, $38, $39
			)
			RETURNING id`,
			transaction.BuyerID,
			transaction.TransactionNo,
			transaction.DownPayment,
			transaction.VoucherID,
			transaction.VoucherType,
			transaction.VoucherAmount,
			transaction.RecipientName,
			transaction.RecipientMobilePhone,
			transaction.BuyerNote,
			transaction.PaymentMethodID,
			transaction.PaymentMethodName,
			transaction.PaymentMethodChannelCode,
			transaction.PaymentMethodExtraData,
			transaction.PaymentNo,
			transaction.PaymentStatus,
			transaction.ShippingStatus,
			transaction.ReviewedBy,
			transaction.ReviewedAt,
			transaction.RejectionReason,
			transaction.ShippingCost,
			transaction.ShippingCostType,
			transaction.WalletDeduction,
			transaction.Cashback,
			transaction.ShippingLongitude,
			transaction.ShippingLatitude,
			transaction.ShippingAddress,
			transaction.ShippingInstruction,
			transaction.ShippingDate,
			transaction.ShippingPlanID,
			transaction.ShippingType,
			transaction.ShippingTimeline,
			transaction.RefundedShippingCost,
			transaction.RefundedTransactionCost,
			transaction.UniqueCode,
			transaction.MinimumPoint,
			transaction.PointDeduction,
			transaction.ProductCategoryDiscountAmount,
			transaction.PromoDiscount,
			transaction.CreatedAt,
		).Scan(&transaction.ID); err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
			Log(slog.LevelError, err.Error())
			return
		}
		mapTransactionIDs[transaction.OldID] = transaction.ID
	}
	// transaction_items
	rows, err = mysql.QueryContext(
		ctx,
		`SELECT
			id
			, idTransaksi
			, idPasarPenjual
			, idPenjual
			, COALESCE(idPromo, 0)
			, diskonPromo
			, idProduk
			, jenisProduk
			, idSatuan
			, harga
			, diskon
			, quantity
			, status
			, statusReason
			, COALESCE(statusDoneBy, 0)
			, statusDoneAt
			, catatan
		FROM transaksi_item
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
	var transactionItems []TransactionItem
	for rows.Next() {
		var transactionItem TransactionItem
		if err = rows.Scan(
			&transactionItem.OldID,
			&transactionItem.OldTransactionID,
			&transactionItem.OldBazaarID,
			&transactionItem.OldSellerID,
			&transactionItem.OldPromoID,
			&transactionItem.PromoDiscount,
			&transactionItem.OldProductID,
			&transactionItem.ProductType,
			&transactionItem.OldProductVariantID,
			&transactionItem.Price,
			&transactionItem.Discount,
			&transactionItem.Quantity,
			&transactionItem.Status,
			&transactionItem.RejectionReason,
			&transactionItem.OldReviewedBy,
			&transactionItem.ReviewedAt,
			&transactionItem.BuyerNote,
		); err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
			Log(slog.LevelError, err.Error())
			return
		}
		transactionID, ok := mapTransactionIDs[transactionItem.OldID]
		if ok {
			transactionItem.TransactionID = &transactionID
		}
		bazaarID, ok := mapBazaarIDs[transactionItem.OldBazaarID]
		if ok {
			transactionItem.BazaarID = bazaarID
		}
		sellerID, ok := mapSellerIDs[transactionItem.OldSellerID]
		if ok {
			transactionItem.SellerID = sellerID
		}
		promoID, ok := mapSellerIDs[transactionItem.OldPromoID]
		if ok {
			transactionItem.PromoID = &promoID
		}
		productID, ok := mapProductIDs[transactionItem.OldProductID]
		if ok {
			transactionItem.ProductID = &productID
		}
		productVariantID, ok := mapProductVariantIDs[transactionItem.OldProductVariantID]
		if ok {
			transactionItem.ProductVariantID = &productVariantID
		}
		reviewedBy, ok := mapAdminIDs[transactionItem.OldReviewedBy]
		if ok {
			transactionItem.ReviewedBy = &reviewedBy
		}
		transactionItems = append(transactionItems, transactionItem)
	}
	for _, transactionItem := range transactionItems {
		if _, err = tx.Exec(
			ctx,
			`INSERT INTO transaction_items (
				transaction_id
				, bazaar_id
				, seller_id
				, product_variant_id
				, product_id
				, product_type
				, promo_id
				, promo_discount
				, price
				, discount
				, quantity
				, status
				, reviewed_by
				, reviewed_at
				, rejection_reason
				, buyer_note
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9
				, $10, $11, $12, $13, $14, $15, $16
			)`,
			transactionItem.TransactionID,
			transactionItem.BazaarID,
			transactionItem.SellerID,
			transactionItem.ProductVariantID,
			transactionItem.ProductID,
			transactionItem.ProductType,
			transactionItem.PromoID,
			transactionItem.PromoDiscount,
			transactionItem.Price,
			transactionItem.Discount,
			transactionItem.Quantity,
			transactionItem.Status,
			transactionItem.ReviewedBy,
			transactionItem.ReviewedAt,
			transactionItem.RejectionReason,
			transactionItem.BuyerNote,
		); err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				Log(slog.LevelError, errRollback.Error())
			}
			Log(slog.LevelError, err.Error())
			return
		}
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
