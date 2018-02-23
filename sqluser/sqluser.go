package sqluser

import (
	"bytes"
	"database/sql"
	"encoding/hex"
	"errors"
	"strconv"
	"time"

	"crypto/rand"
	"crypto/sha256"

	"github.com/herb-go/herb/model/sql/datamapper"
	"github.com/herb-go/herb/model/sql/query"
	"github.com/herb-go/herb/user"
	"github.com/herb-go/member"
	"github.com/satori/go.uuid"
)

const (
	//FlagEmpty sql user create flag empty
	FlagEmpty = 0
	//FlagWithAccount sql user create flag with account module
	FlagWithAccount = 1
	//FlagWithPassword sql user create flag with password module
	FlagWithPassword = 2
	//FlagWithToken sql user create flag with token module
	FlagWithToken = 4
	//FlagWithUser sql user create flag with user module
	FlagWithUser = 8
)

const (
	//UserStatusNormal normal status in user module
	UserStatusNormal = 0
	//UserStatusBanned baneed status in user module
	UserStatusBanned = 1
)

//RandomBytesLength bytes length for RandomBytes function.
var RandomBytesLength = 32

//ErrHashMethodNotFound error raised when password hash method not found.
var ErrHashMethodNotFound = errors.New("password hash method not found")

//HashFunc interaface of pasword hash func
type HashFunc func(key string, salt string, password string) ([]byte, error)

//DefaultAccountTableName default database table name for module account.
var DefaultAccountTableName = "account"

//DefaultPasswordTableName default database table name for module password.
var DefaultPasswordTableName = "password"

//DefaultTokenTableName default database table name for module token.
var DefaultTokenTableName = "token"

//DefaultUserTableName default database table name for module user.
var DefaultUserTableName = "user"

//DefaultHashMethod default hash method when created password data.
var DefaultHashMethod = "sha256"

//HashFuncMap all available password hash func.
//You can insert custom hash func into this map.
var HashFuncMap = map[string]HashFunc{
	"sha256": func(key string, salt string, password string) ([]byte, error) {
		var val = []byte(key + salt + password)
		var s256 = sha256.New()
		s256.Write(val)
		val = s256.Sum(nil)
		s256.Write(val)
		return []byte(hex.EncodeToString(s256.Sum(nil))), nil
	},
}

//New create User framework with given database setting and falg.
//flag is values combine with flags to special which modules used.
//For example ,New(db,FlagWithAccount | FlagWithToken)
func New(db datamapper.DB, flag int) *User {
	return &User{
		DB: db,
		Tables: Tables{
			AccountTableName:  DefaultAccountTableName,
			PasswordTableName: DefaultPasswordTableName,
			TokenTableName:    DefaultTokenTableName,
			UserTableName:     DefaultUserTableName,
		},
		HashMethod:     DefaultHashMethod,
		UIDGenerater:   UUID,
		TokenGenerater: Timestamp,
		SaltGenerater:  RandomBytes,
		Flag:           flag,
	}
}

//Tables struct stores table info.
type Tables struct {
	AccountTableName  string
	PasswordTableName string
	TokenTableName    string
	UserTableName     string
}

//RandomBytes string generater return random bytes.
//Default length is 32 byte.You can change default length by change sqluesr.RandomBytesLength .
func RandomBytes() (string, error) {
	var bytes = make([]byte, RandomBytesLength)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

//UUID string generater return UUID.
func UUID() (string, error) {
	return uuid.NewV1().String(), nil
}

//Timestamp string generater return timestamp in nano.
func Timestamp() (string, error) {
	return strconv.FormatInt(time.Now().UnixNano(), 10), nil
}

//User main struct of sqluser module.
type User struct {
	//DB database used.
	DB datamapper.DB
	//Tables table name info.
	Tables Tables
	//Flag sqluser modules create flg.
	Flag int
	//UIDGenerater string generater for uid
	//default value is uuid
	UIDGenerater func() (string, error)
	//TokenGenerater string generater for usertoken
	//default value is timestamp
	TokenGenerater func() (string, error)
	//SaltGenerater string generater for salt
	//default value is 32 byte length random bytes.
	SaltGenerater func() (string, error)
	//HashMethod hash method which used to generate new salt.
	//default value is sha256
	HashMethod string
	//PasswordKey static key used in passwrod hash generater.
	//default value is empty.
	//You can change this value after sqluser init.
	PasswordKey string
}

//HasFlag check if sqluser module created with special flag.
func (u *User) HasFlag(flag int) bool {
	return u.Flag&flag != 0
}

//AccountTableName return actual account database table name.
func (u *User) AccountTableName() string {
	return u.DB.TableName(u.Tables.AccountTableName)
}

//PasswordTableName return actual password database table name.
func (u *User) PasswordTableName() string {
	return u.DB.TableName(u.Tables.PasswordTableName)
}

//TokenTableName return actual token database table name.
func (u *User) TokenTableName() string {
	return u.DB.TableName(u.Tables.TokenTableName)
}

//UserTableName return actual user database table name.
func (u *User) UserTableName() string {
	return u.DB.TableName(u.Tables.UserTableName)
}

//Account return account data mapper
func (u *User) Account() *AccountDataMapper {
	return &AccountDataMapper{
		DataMapper: datamapper.New(u.DB, u.Tables.AccountTableName),
		User:       u,
	}
}

//Password return password data mapper
func (u *User) Password() *PasswordDataMapper {
	return &PasswordDataMapper{
		DataMapper: datamapper.New(u.DB, u.Tables.PasswordTableName),
		User:       u,
	}
}

//Token return token data mapper
func (u *User) Token() *TokenDataMapper {
	return &TokenDataMapper{
		DataMapper: datamapper.New(u.DB, u.Tables.TokenTableName),
		User:       u,
	}
}

//User return user data mapper
func (u *User) User() *UserDataMapper {
	return &UserDataMapper{
		DataMapper: datamapper.New(u.DB, u.Tables.UserTableName),
		User:       u,
	}
}

//AccountDataMapper account data mapper
type AccountDataMapper struct {
	datamapper.DataMapper
	User    *User
	Service *member.Service
}

//InstallToMember install account module to member service as provider
func (a *AccountDataMapper) InstallToMember(service *member.Service) {
	service.AccountsProvider = a
	a.Service = service
}

//Unbind unbind account from user.
//Return any error if raised.
func (a *AccountDataMapper) Unbind(uid string, account *user.Account) error {
	tx, err := a.DB().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	Delete := query.NewDelete(a.DBTableName())
	Delete.Where.Condition = query.And(
		query.Equal("account.uid", uid),
		query.Equal("account.keyword", account.Keyword),
		query.Equal("account.account", account.Account),
	)
	_, err = Delete.Query().Exec(tx)
	if err != nil {
		return err
	}
	return tx.Commit()

}

//Bind bind account to user.
//Return any error if raised.
//If account exists, error user.ErrAccountBindExists will raised.
func (a *AccountDataMapper) Bind(uid string, account *user.Account) error {
	tx, err := a.DB().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var u = ""
	Select := query.NewSelect()
	Select.Select.Add("account.uid")
	Select.From.AddAlias("account", a.DBTableName())
	Select.Where.Condition = query.And(
		query.Equal("keyword", account.Keyword),
		query.Equal("account", account.Account),
	)
	row := Select.QueryRow(a.DB())
	err = row.Scan(&u)
	if err != nil {
		if err != sql.ErrNoRows {
			return err
		}
	} else {
		return user.ErrAccountBindExists

	}

	var CreatedTime = time.Now().Unix()
	Insert := query.NewInsert(a.DBTableName())
	Insert.Insert.
		Add("uid", uid).
		Add("keyword", account.Keyword).
		Add("account", account.Account).
		Add("created_time", CreatedTime)
	_, err = Insert.Query().Exec(tx)
	if err != nil {
		return err
	}
	return tx.Commit()
}

//FindOrInsert find user by account.if account did not exists,a new user with given account will be created.
//UIDGenerater used when create new user.
//Return user id and any error if raised.
func (a *AccountDataMapper) FindOrInsert(UIDGenerater func() (string, error), account *user.Account) (string, error) {
	var result = AccountModel{}
	tx, err := a.DB().Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	Select := query.NewSelect()
	Select.From.AddAlias("account", a.DBTableName())
	Select.Select.Add("account.uid", "account.keyword", "account.account", "account.created_time")
	Select.Where.Condition = query.And(
		query.Equal("account.keyword", account.Keyword),
		query.Equal("account.account", account.Account),
	)
	row := Select.QueryRow(a.DB())
	err = Select.Result().
		Bind("account.uid", &result.UID).
		Bind("account.keyword", &result.Keyword).
		Bind("account.account", &result.Account).
		Bind("account.created_time", &result.CreatedTime).
		ScanFrom(row)
	if err == nil {
		return result.UID, nil
	}
	if err != sql.ErrNoRows {
		return "", err
	}
	uid, err := UIDGenerater()
	var CreatedTime = time.Now().Unix()
	Insert := query.NewInsert(a.DBTableName())
	Insert.Insert.
		Add("uid", uid).
		Add("keyword", account.Keyword).
		Add("account", account.Account).
		Add("created_time", CreatedTime)
	_, err = Insert.Query().Exec(tx)
	if err != nil {
		return "", err
	}
	if a.User.HasFlag(FlagWithUser) {
		Insert := query.NewInsert(a.User.UserTableName())
		Insert.Insert.
			Add("uid", uid).
			Add("status", UserStatusNormal).
			Add("created_time", CreatedTime).
			Add("updated_time", CreatedTime)
		_, err = Insert.Query().Exec(tx)
		if err != nil {
			return "", err
		}
	}
	return uid, tx.Commit()
}

//Insert create new user with given account.
//Return any error if raised.
//If account exists,member.ErrAccountRegisterExists will raise.
func (a *AccountDataMapper) Insert(uid string, keyword string, account string) error {
	tx, err := a.DB().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var u = ""
	Select := query.NewSelect()
	Select.Select.Add("uid")
	Select.From.Add(a.DBTableName())
	Select.Where.Condition = query.And(
		query.Equal("keyword", keyword),
		query.Equal("account", account),
	)
	row := Select.QueryRow(a.DB())
	err = row.Scan(&u)
	if err != nil {
		if err != sql.ErrNoRows {
			return err
		}
	} else {
		return member.ErrAccountRegisterExists
	}
	var CreatedTime = time.Now().Unix()
	Insert := query.NewInsert(a.DBTableName())
	Insert.Insert.
		Add("uid", uid).
		Add("keyword", keyword).
		Add("account", account).
		Add("created_time", CreatedTime)
	_, err = Insert.Query().Exec(tx)
	if err != nil {
		return err
	}
	if a.User.HasFlag(FlagWithUser) {
		Insert := query.NewInsert(a.User.UserTableName())
		Insert.Insert.
			Add("uid", uid).
			Add("status", UserStatusNormal).
			Add("created_time", CreatedTime).
			Add("updated_time", CreatedTime)
		_, err = Insert.Query().Exec(tx)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

//Find find account by given keyword and account.
//Return account model and any error if raised.
func (a *AccountDataMapper) Find(keyword string, account string) (AccountModel, error) {
	var result = AccountModel{}
	if keyword == "" || account == "" {
		return result, sql.ErrNoRows
	}
	Select := query.NewSelect()
	Select.Select.Add("uid", "keyword", "account", "created_time")
	Select.From.Add(a.DBTableName())
	Select.Where.Condition = query.And(
		query.Equal("keyword", keyword),
		query.Equal("account", account),
	)
	row := Select.QueryRow(a.DB())
	err := Select.Result().
		Bind("uid", &result.UID).
		Bind("keyword", &result.Keyword).
		Bind("account", &result.Account).
		Bind("created_time", &result.CreatedTime).
		ScanFrom(row)
	return result, err
}

//FindAllByUID find account models by user id list.
//Retrun account models and any error if rased.
func (a *AccountDataMapper) FindAllByUID(uids ...string) ([]AccountModel, error) {
	var result = []AccountModel{}
	if len(uids) == 0 {
		return result, nil
	}
	Select := query.NewSelect()
	Select.Select.Add("account.uid", "account.keyword", "account.account")
	Select.From.AddAlias("account", a.DBTableName())
	Select.Where.Condition = query.In("account.uid", uids)
	rows, err := Select.QueryRows(a.DB())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		v := AccountModel{}
		err := Select.Result().
			Bind("account.uid", &v.UID).
			Bind("account.keyword", &v.Keyword).
			Bind("account.account", &v.Account).
			ScanFrom(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, nil
}

//Accounts get member account map by user id list.
//Return account map and any error if rasied.
//User unfound in account map will be a nil value.
func (a *AccountDataMapper) Accounts(uid ...string) (member.Accounts, error) {
	models, err := a.FindAllByUID(uid...)
	if err != nil {
		return nil, err
	}
	result := member.Accounts{}
	for _, v := range models {
		if result[v.UID] == nil {
			result[v.UID] = user.Accounts{}
		}
		account := user.Account{Keyword: v.Keyword, Account: v.Account}
		result[v.UID] = append(result[v.UID], &account)
	}
	return result, nil
}

//AccountToUID find user by account.
//Return user id and any error if rasied.
//If user not found,a empty string will be returned.
func (a *AccountDataMapper) AccountToUID(account *user.Account) (uid string, err error) {
	model, err := a.Find(account.Keyword, account.Account)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return model.UID, err
}

//Register register a user with special account.
//Return user id and any error if raised.
//If account exists,member.ErrAccountRegisterExists will raise.
func (a *AccountDataMapper) Register(account *user.Account) (uid string, err error) {
	uid, err = a.User.UIDGenerater()
	if err != nil {
		return
	}
	err = a.Insert(uid, account.Keyword, account.Account)
	return
}

//AccountToUIDOrRegister find a user by account.if user didnot exist,a new user will be created.
//Return user id and any error if raised.
func (a *AccountDataMapper) AccountToUIDOrRegister(account *user.Account) (uid string, err error) {
	return a.FindOrInsert(a.User.UIDGenerater, account)
}

//BindAccount bind account to user.
//Return any error if rasied.
//If account exists, error user.ErrAccountBindExists will raised.
func (a *AccountDataMapper) BindAccount(uid string, account *user.Account) error {
	return a.Bind(uid, account)
}

//UnbindAccount unbind account from user.
//Return any error if rasied.
func (a *AccountDataMapper) UnbindAccount(uid string, account *user.Account) error {
	return a.Unbind(uid, account)
}

//AccountModel account data model
type AccountModel struct {
	//UID user id.
	UID string
	//Keyword account keyword.
	Keyword string
	//Account account name.
	Account string
	//CreatedTime created timestamp in second.
	CreatedTime int64
}

//PasswordDataMapper password data mapper
type PasswordDataMapper struct {
	datamapper.DataMapper
	User    *User
	Service *member.Service
}

//InstallToMember install passowrd module to member service as provider
func (p *PasswordDataMapper) InstallToMember(service *member.Service) {
	service.PasswordProvider = p
	p.Service = service
}

//Find find password model by userd id.
//Return any error if raised.
func (p *PasswordDataMapper) Find(uid string) (PasswordModel, error) {
	var result = PasswordModel{}
	if uid == "" {
		return result, sql.ErrNoRows
	}
	Select := query.NewSelect()
	Select.Select.Add("password.hash_method", "password.salt", "password.password", "password.updated_time")
	Select.From.AddAlias("password", p.DBTableName())
	Select.Where.Condition = query.Equal("uid", uid)
	q := Select.Query()
	row := p.DB().QueryRow(q.QueryCommand(), q.QueryArgs()...)
	result.UID = uid
	args := Select.Result().
		Bind("password.hash_method", &result.HashMethod).
		Bind("password.salt", &result.Salt).
		Bind("password.password", &result.Password).
		Bind("password.updated_time", &result.UpdatedTime).
		Args()
	err := row.Scan(args...)
	return result, err
}

//InsertOrUpdate insert or update password model.
//Return any error if raised.
func (p *PasswordDataMapper) InsertOrUpdate(model *PasswordModel) error {
	tx, err := p.DB().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	Update := query.NewUpdate(p.DBTableName())
	Update.Update.
		Add("hash_method", model.HashMethod).
		Add("salt", model.Salt).
		Add("password", model.Password).
		Add("updated_time", model.UpdatedTime)
	Update.Where.Condition = query.Equal("uid", model.UID)
	r, err := Update.Query().Exec(tx)

	if err != nil {
		return err
	}
	affected, err := r.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 0 {
		return tx.Commit()
	}
	Insert := query.NewInsert(p.DBTableName())
	Insert.Insert.
		Add("uid", model.UID).
		Add("hash_method", model.HashMethod).
		Add("salt", model.Salt).
		Add("password", model.Password).
		Add("updated_time", model.UpdatedTime)
	_, err = Insert.Query().Exec(tx)
	if err != nil {
		return err
	}
	return tx.Commit()
}

//VerifyPassword Verify user password.
//Return verify and any error if raised.
//if user not found,error member.ErrUserNotFound will be raised.
func (p *PasswordDataMapper) VerifyPassword(uid string, password string) (bool, error) {
	model, err := p.Find(uid)
	if err == sql.ErrNoRows {
		return false, member.ErrUserNotFound
	}
	if err != nil {
		return false, err
	}
	hash := HashFuncMap[model.HashMethod]
	if hash == nil {
		return false, ErrHashMethodNotFound
	}
	hashed, err := hash(p.User.PasswordKey, model.Salt, password)
	if err != nil {
		return false, err
	}
	return bytes.Compare(hashed, model.Password) == 0, nil
}

//UpdatePassword update user password.If user password does not exist,new password record will be created.
//Return any error if raised.
func (p *PasswordDataMapper) UpdatePassword(uid string, password string) error {
	salt, err := p.User.SaltGenerater()
	if err != nil {
		return err
	}
	hash := HashFuncMap[p.User.HashMethod]
	if hash == nil {
		return ErrHashMethodNotFound
	}
	hashed, err := hash(p.User.PasswordKey, salt, password)
	if err != nil {
		return err
	}
	model := &PasswordModel{
		UID:         uid,
		HashMethod:  p.User.HashMethod,
		Salt:        salt,
		Password:    hashed,
		UpdatedTime: time.Now().Unix(),
	}
	return p.InsertOrUpdate(model)
}

//PasswordModel password data model
type PasswordModel struct {
	//UID user id.
	UID string
	//HashMethod hash method to verify this password.
	HashMethod string
	//Salt random salt.
	Salt string
	//Password hashed password data.
	Password []byte
	//UpdatedTime updated timestamp in second.
	UpdatedTime int64
}

//TokenDataMapper token data mapper
type TokenDataMapper struct {
	datamapper.DataMapper
	User    *User
	Service *member.Service
}

//InstallToMember install token module to member service as provider
func (t *TokenDataMapper) InstallToMember(service *member.Service) {
	service.TokenProvider = t
	t.Service = service
}

//InsertOrUpdate insert or update user token record.
func (t *TokenDataMapper) InsertOrUpdate(uid string, token string) error {
	tx, err := t.DB().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var CreatedTime = time.Now().Unix()
	Update := query.NewUpdate(t.DBTableName())
	Update.Update.
		Add("token", token).
		Add("updated_time", CreatedTime)
	Update.Where.Condition = query.Equal("uid", uid)
	r, err := Update.Query().Exec(tx)
	if err != nil {
		return err
	}
	affected, err := r.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 0 {
		return tx.Commit()
	}
	Insert := query.NewInsert(t.DBTableName())
	Insert.Insert.
		Add("uid", uid).
		Add("token", token).
		Add("updated_time", CreatedTime)
	_, err = Insert.Query().Exec(tx)
	if err != nil {
		return err
	}
	return tx.Commit()
}

//FindAllByUID find all token model by uid list.
//Return token models and any error if raised.
func (t *TokenDataMapper) FindAllByUID(uids ...string) ([]TokenModel, error) {
	var result = []TokenModel{}
	if len(uids) == 0 {
		return result, nil
	}
	Select := query.NewSelect()
	Select.Select.Add("token.uid", "token.token")
	Select.From.AddAlias("token", t.DBTableName())
	Select.Where.Condition = query.In("token.uid", uids)
	rows, err := Select.QueryRows(t.DB())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		v := TokenModel{}
		err = rows.Scan(&v.UID, &v.Token)
		if err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, nil
}

//Tokens get member token map by user id list.
//Return token map and any error if rasied.
//User unfound in token map will be a nil value.
func (t *TokenDataMapper) Tokens(uid ...string) (member.Tokens, error) {
	models, err := t.FindAllByUID(uid...)
	if err != nil {
		return nil, err
	}
	result := member.Tokens{}
	for _, v := range models {
		result[v.UID] = v.Token
	}
	return result, nil

}

//Revoke revoke and regenerate a new token to user.if revoke record does not exist,a new record will be created.
//Return new user token and any error if raised.
func (t *TokenDataMapper) Revoke(uid string) (string, error) {
	token, err := t.User.TokenGenerater()
	if err != nil {
		return "", err
	}
	return token, t.InsertOrUpdate(uid, token)
}

//TokenModel token data model
type TokenModel struct {
	//UID user id
	UID string
	//Token current user token
	Token string
	//UpdatedTime updated timestamp in second.
	UpdatedTime string
}

//UserDataMapper user data mapper
type UserDataMapper struct {
	datamapper.DataMapper
	User    *User
	Service *member.Service
}

//InstallToMember install user module to member service as provider
func (u *UserDataMapper) InstallToMember(service *member.Service) {
	service.BannedProvider = u
	u.Service = service
}

//FindAllByUID find user models by user id list.
//Return User model list and any error if raised.
func (u *UserDataMapper) FindAllByUID(uids ...string) ([]UserModel, error) {
	var result = []UserModel{}
	if len(uids) == 0 {
		return result, nil
	}
	Select := query.NewSelect()
	Select.Select.Add("user.uid", "user.status")
	Select.From.AddAlias("user", u.DBTableName())
	Select.Where.Condition = query.In("user.uid", uids)
	rows, err := Select.QueryRows(u.DB())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		v := UserModel{}
		err = rows.Scan(&v.UID, &v.Status)
		if err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, nil
}

//InsertOrUpdate insert or update user model with status.
//Return any error if raised.
func (u *UserDataMapper) InsertOrUpdate(uid string, status int) error {
	tx, err := u.DB().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var CreatedTime = time.Now().Unix()
	Update := query.NewUpdate(u.DBTableName())
	Update.Update.
		Add("status", status).
		Add("updated_time", CreatedTime)
	Update.Where.Condition = query.Equal("uid", uid)
	r, err := Update.Query().Exec(tx)
	if err != nil {
		return err
	}
	affected, err := r.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 0 {
		return tx.Commit()
	}
	Insert := query.NewInsert(u.DBTableName())
	Insert.Insert.
		Add("uid", uid).
		Add("status", status).
		Add("updated_time", CreatedTime).
		Add("created_time", CreatedTime)
	_, err = Insert.Query().Exec(tx)
	if err != nil {
		return err
	}
	return tx.Commit()
}

//Banned get member banned status map by user id list.
//Return banned status map and any error if rasied.
//User unfound in token map will be false.
func (u *UserDataMapper) Banned(uid ...string) (member.BannedMap, error) {
	models, err := u.FindAllByUID(uid...)
	if err != nil {
		return nil, err
	}
	result := member.BannedMap{}
	for _, v := range models {
		result[v.UID] = (v.Status == UserStatusBanned)
	}
	return result, nil
}

//Ban set user banned status.
//Return any error if raised.
func (u *UserDataMapper) Ban(uid string, banned bool) error {
	var status int
	if banned {
		status = UserStatusBanned
	} else {
		status = UserStatusNormal
	}
	return u.InsertOrUpdate(uid, status)

}

//UserModel user data model
type UserModel struct {
	//UID user id
	UID string
	//CreatedTime created timestamp in second
	CreatedTime int64
	//UpdateTIme updated timestamp in second
	UpdateTIme int64
	//Status user status
	Status int
}
