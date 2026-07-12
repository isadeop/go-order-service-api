package custom_errors

import "errors"

var ErrClientNotFound = errors.New("client not found / cliente não encontrado")
var ErrClientNameRequired = errors.New("client name is required / nome do cliente é obrigatório")
var ErrClientNameTooShort = errors.New("client name must have at least 3 characters / nome do cliente deve ter pelo menos 3 caracteres")
var ErrClientNameTooLong = errors.New("client name must have at most 255 characters / nome do cliente deve ter no máximo 255 caracteres")
var ErrClientEmailRequired = errors.New("client email is required / email do cliente é obrigatório")
var ErrClientEmailInvalid = errors.New("client email must be valid / email do cliente deve ser válido")
var ErrClientEmailTooLong = errors.New("client email must have at most 255 characters / email do cliente deve ter no máximo 255 caracteres")
var ErrClientPhoneRequired = errors.New("client phone is required / telefone do cliente é obrigatório")
var ErrClientPhoneInvalid = errors.New("client phone must be valid / telefone do cliente deve ser válido")
var ErrClientPhoneTooLong = errors.New("client phone must have at most 30 characters / telefone do cliente deve ter no máximo 30 caracteres")
var ErrClientEmailAlreadyExists = errors.New("client email already exists / email do cliente já existe")
var ErrInvalidClientID = errors.New("client id must be a valid uuid / id do cliente deve ser um uuid válido")
var ErrClientPasswordRequired = errors.New("client password is required / senha do cliente é obrigatória")
var ErrClientPasswordTooShort = errors.New("client password must have at least 8 characters / senha do cliente deve ter pelo menos 8 caracteres")
var ErrClientPasswordTooLong = errors.New("client password is too long / senha do cliente é muito longa")
var ErrClientPasswordHashRequired = errors.New("client password hash is required / hash da senha do cliente é obrigatório")
var ErrInvalidCredentials = errors.New("invalid credentials / credenciais inválidas")
