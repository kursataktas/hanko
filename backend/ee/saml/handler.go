package saml

import (
	"errors"
	"fmt"
	"github.com/gobuffalo/pop/v6"
	"github.com/labstack/echo/v4"
	saml2 "github.com/russellhaering/gosaml2"
	auditlog "github.com/teamhanko/hanko/backend/audit_log"
	"github.com/teamhanko/hanko/backend/config"
	samlConfig "github.com/teamhanko/hanko/backend/ee/saml/config"
	"github.com/teamhanko/hanko/backend/ee/saml/dto"
	"github.com/teamhanko/hanko/backend/ee/saml/provider"
	samlUtils "github.com/teamhanko/hanko/backend/ee/saml/utils"
	"github.com/teamhanko/hanko/backend/persistence"
	"github.com/teamhanko/hanko/backend/persistence/models"
	"github.com/teamhanko/hanko/backend/session"
	"github.com/teamhanko/hanko/backend/thirdparty"
	"github.com/teamhanko/hanko/backend/utils"
	"golang.org/x/exp/slices"
	"net/http"
	"net/url"
	"strings"
)

type SamlHandler struct {
	auditLogger    auditlog.Logger
	config         *config.Config
	persister      persistence.Persister
	sessionManager session.Manager
	providers      []provider.ServiceProvider
}

const (
	unableToLoadProviderError = "unable to load providers"
	metadataErrorMessage      = "unable to provide metadata"
)

func NewSamlHandler(cfg *config.Config, persister persistence.Persister, sessionManager session.Manager, auditLogger auditlog.Logger) *SamlHandler {
	providers := make([]provider.ServiceProvider, 0)
	for _, idpConfig := range cfg.Saml.IdentityProviders {
		if idpConfig.Enabled {
			newProvider, err := initializeServiceProvider(idpConfig, cfg, persister)
			if err != nil {
				panic(err)
			}

			providers = append(providers, *newProvider)
		}
	}

	return &SamlHandler{
		auditLogger:    auditLogger,
		config:         cfg,
		persister:      persister,
		sessionManager: sessionManager,
		providers:      providers,
	}
}

func initializeServiceProvider(idpConfig samlConfig.IdentityProvider, cfg *config.Config, persister persistence.Persister) (*provider.ServiceProvider, error) {
	name := ""
	name, err := parseProviderFromMetadataUrl(idpConfig.MetadataUrl)
	if err != nil {
		return nil, err
	}

	newProvider, err := provider.GetProvider(name, cfg, idpConfig, persister.GetSamlCertificatePersister())
	if err != nil {
		return nil, err
	}

	return &newProvider, nil
}

func parseProviderFromMetadataUrl(idpUrlString string) (string, error) {
	idpUrl, err := url.Parse(idpUrlString)
	if err != nil {
		return "", err
	}

	return idpUrl.Host, nil
}

func (handler *SamlHandler) getProviderByDomain(domain string, providers []provider.ServiceProvider) (provider.ServiceProvider, error) {
	if len(providers) == 0 {
		return nil, errors.New("no provider configured")
	}

	for _, availableProvider := range providers {
		if availableProvider.GetDomain() == domain {
			return availableProvider, nil
		}
	}

	return nil, fmt.Errorf("unknown provider for domain %s", domain)
}

func (handler *SamlHandler) Metadata(c echo.Context) error {
	var request dto.SamlMetadataRequest
	err := c.Bind(&request)
	if err != nil {
		c.Logger().Error(err)
		return c.JSON(http.StatusBadRequest, thirdparty.ErrorInvalidRequest("domain is missing"))
	}

	providerList, err := handler.addDbProviders(c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, thirdparty.ErrorServer(metadataErrorMessage).WithCause(err))
	}

	foundProvider, err := handler.getProviderByDomain(request.Domain, providerList)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusNotFound, err)
	}

	if request.CertOnly {
		cert, err := handler.persister.GetSamlCertificatePersister().GetFirst()
		if err != nil {
			c.Logger().Error(err)
			return c.JSON(http.StatusInternalServerError, thirdparty.ErrorServer(metadataErrorMessage).WithCause(err))
		}

		if cert == nil {
			return c.NoContent(http.StatusNotFound)
		}

		c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename=%s-service-provider.pem", handler.config.Service.Name))
		return c.Blob(http.StatusOK, echo.MIMEOctetStream, []byte(cert.CertData))
	}

	xmlMetadata, err := foundProvider.ProvideMetadataAsXml()
	if err != nil {
		c.Logger().Error(err)
		return c.JSON(http.StatusInternalServerError, thirdparty.ErrorServer(metadataErrorMessage).WithCause(err))
	}

	c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename=%s-metadata.xml", handler.config.Service.Name))
	return c.Blob(http.StatusOK, echo.MIMEOctetStream, xmlMetadata)
}

func (handler *SamlHandler) Auth(c echo.Context) error {
	errorRedirectTo := c.Request().Header.Get("Referer")
	if errorRedirectTo == "" {
		errorRedirectTo = handler.config.Saml.DefaultRedirectUrl
	}

	var request dto.SamlAuthRequest
	err := c.Bind(&request)
	if err != nil {
		c.Logger().Error(err)
		return handler.redirectError(c, thirdparty.ErrorInvalidRequest(err.Error()).WithCause(err), errorRedirectTo)
	}

	err = c.Validate(request)
	if err != nil {
		c.Logger().Error(err)
		return handler.redirectError(c, thirdparty.ErrorInvalidRequest(err.Error()).WithCause(err), errorRedirectTo)
	}

	if ok := samlUtils.IsAllowedRedirect(handler.config.Saml, request.RedirectTo); !ok {
		return handler.redirectError(c, thirdparty.ErrorInvalidRequest(fmt.Sprintf("redirect to '%s' not allowed", request.RedirectTo)), errorRedirectTo)
	}

	providerList, err := handler.addDbProviders(c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, thirdparty.ErrorServer(unableToLoadProviderError).WithCause(err))
	}

	foundProvider, err := handler.getProviderByDomain(request.Domain, providerList)
	if err != nil {
		c.Logger().Error(err)
		return handler.redirectError(c, thirdparty.ErrorInvalidRequest(err.Error()).WithCause(err), errorRedirectTo)
	}

	state, err := GenerateState(
		handler.config,
		handler.persister.GetSamlStatePersister(),
		request.Domain,
		request.RedirectTo)

	if err != nil {
		c.Logger().Error(err)
		return handler.redirectError(c, thirdparty.ErrorServer("could not generate state").WithCause(err), errorRedirectTo)
	}

	redirectUrl, err := foundProvider.GetService().BuildAuthURL(string(state))
	if err != nil {
		c.Logger().Error(err)
		return handler.redirectError(c, thirdparty.ErrorServer("could not generate auth url").WithCause(err), errorRedirectTo)
	}

	return c.Redirect(http.StatusTemporaryRedirect, redirectUrl)
}

func (handler *SamlHandler) CallbackPost(c echo.Context) error {
	state, samlError := VerifyState(handler.config, handler.persister.GetSamlStatePersister(), c.FormValue("RelayState"))
	if samlError != nil {
		c.Logger().Error(samlError)
		return handler.redirectError(
			c,
			thirdparty.ErrorInvalidRequest(samlError.Error()).WithCause(samlError),
			handler.config.Saml.DefaultRedirectUrl,
		)
	}

	if strings.TrimSpace(state.RedirectTo) == "" {
		state.RedirectTo = handler.config.Saml.DefaultRedirectUrl
	}

	redirectTo, samlError := url.Parse(state.RedirectTo)
	if samlError != nil {
		c.Logger().Error(samlError)
		return handler.redirectError(
			c,
			thirdparty.ErrorServer("unable to parse redirect url").WithCause(samlError),
			handler.config.Saml.DefaultRedirectUrl,
		)
	}

	providerList, err := handler.addDbProviders(c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, thirdparty.ErrorServer(unableToLoadProviderError).WithCause(err))
	}

	foundProvider, err := handler.getProviderByDomain(state.Provider, providerList)
	if err != nil {
		c.Logger().Error(err)
		return handler.redirectError(
			c,
			thirdparty.ErrorServer("unable to find provider by domain").WithCause(err),
			redirectTo.String(),
		)
	}

	assertionInfo, samlError := handler.parseSamlResponse(foundProvider, c.FormValue("SAMLResponse"))
	if samlError != nil {
		c.Logger().Error(samlError)
		return handler.redirectError(
			c,
			thirdparty.ErrorServer("unable to parse saml response").WithCause(samlError),
			redirectTo.String(),
		)
	}

	redirectUrl, samlError := handler.linkAccount(c, redirectTo, state, foundProvider, assertionInfo)
	if samlError != nil {
		c.Logger().Error(samlError)
		return handler.redirectError(
			c,
			samlError,
			redirectTo.String(),
		)
	}

	return c.Redirect(http.StatusFound, redirectUrl.String())
}

func (handler *SamlHandler) linkAccount(c echo.Context, redirectTo *url.URL, state *State, provider provider.ServiceProvider, assertionInfo *saml2.AssertionInfo) (*url.URL, error) {
	var accountLinkingResult *thirdparty.AccountLinkingResult
	var samlError error
	samlError = handler.persister.Transaction(func(tx *pop.Connection) error {
		userdata := provider.GetUserData(assertionInfo)

		linkResult, samlError := thirdparty.LinkAccount(tx, handler.config, handler.persister, userdata, state.Provider)
		if samlError != nil {
			return samlError
		}
		accountLinkingResult = linkResult

		token, samlError := handler.createHankoToken(linkResult, tx)
		if samlError != nil {
			return samlError
		}

		query := redirectTo.Query()
		query.Add(utils.HankoTokenQuery, token.Value)
		redirectTo.RawQuery = query.Encode()

		cookie := utils.GenerateStateCookie(handler.config, utils.HankoThirdpartyStateCookie, "", utils.CookieOptions{
			MaxAge:   -1,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
		})
		c.SetCookie(cookie)

		return nil

	})

	if samlError != nil {
		return nil, samlError
	}

	samlError = handler.auditLogger.Create(c, accountLinkingResult.Type, accountLinkingResult.User, nil)

	if samlError != nil {
		return nil, samlError
	}

	return redirectTo, nil
}

func (handler *SamlHandler) createHankoToken(linkResult *thirdparty.AccountLinkingResult, tx *pop.Connection) (*models.Token, error) {
	token, tokenError := models.NewToken(linkResult.User.ID)
	if tokenError != nil {
		return nil, thirdparty.ErrorServer("could not create token").WithCause(tokenError)
	}

	tokenError = handler.persister.GetTokenPersisterWithConnection(tx).Create(*token)
	if tokenError != nil {
		return nil, thirdparty.ErrorServer("could not save token to db").WithCause(tokenError)
	}

	return token, nil
}

func (handler *SamlHandler) parseSamlResponse(provider provider.ServiceProvider, samlResponse string) (*saml2.AssertionInfo, error) {
	assertionInfo, err := provider.GetService().RetrieveAssertionInfo(samlResponse)
	if err != nil {
		return nil, thirdparty.ErrorInvalidRequest("unable to parse SAML response").WithCause(err)
	}

	if assertionInfo.WarningInfo.InvalidTime {
		return nil, thirdparty.ErrorInvalidRequest("SAMLAssertion expired")
	}

	if assertionInfo.WarningInfo.NotInAudience {
		return nil, thirdparty.ErrorInvalidRequest("not in SAML audience")
	}

	return assertionInfo, nil
}

func (handler *SamlHandler) redirectError(c echo.Context, error error, to string) error {
	err := handler.auditError(c, error)
	if err != nil {
		error = err
	}

	redirectURL := thirdparty.GetErrorUrl(to, error)
	return c.Redirect(http.StatusSeeOther, redirectURL)
}

func (handler *SamlHandler) auditError(c echo.Context, err error) error {
	var e *thirdparty.ThirdPartyError
	ok := errors.As(err, &e)

	var auditLogError error
	if ok && e.Code != thirdparty.ErrorCodeServerError {
		auditLogError = handler.auditLogger.Create(c, models.AuditLogThirdPartySignInSignUpFailed, nil, err)
	}
	return auditLogError
}

func (handler *SamlHandler) GetProvider(c echo.Context) error {
	var request dto.SamlRequest
	err := c.Bind(&request)
	if err != nil {
		c.Logger().Error(err)
		return c.JSON(http.StatusBadRequest, err)
	}

	providerList, err := handler.addDbProviders(c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, thirdparty.ErrorServer(unableToLoadProviderError).WithCause(err))
	}

	foundProvider, err := handler.getProviderByDomain(request.Domain, providerList)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusNotFound, err)
	}

	return c.JSON(http.StatusOK, foundProvider.GetConfig())
}

func (handler *SamlHandler) addDbProviders(ctx echo.Context) ([]provider.ServiceProvider, error) {
	serviceProviders := handler.providers

	dbProviders, err := handler.persister.GetSamlIdentityProviderPersister(nil).List()
	if err != nil {
		ctx.Logger().Error(err)
		return nil, err
	}

	for _, dbProvider := range dbProviders {
		if dbProvider.Enabled {
			isAlreadyRegistered := slices.ContainsFunc(handler.providers, func(idp provider.ServiceProvider) bool {
				return idp.GetDomain() == dbProvider.Domain
			})

			if isAlreadyRegistered {
				ctx.Logger().Warn("Provider with domain is already registered from config file")
				continue
			}

			attributeMap := samlConfig.AttributeMap{
				Name:              dbProvider.AttributeMap.Name,
				FamilyName:        dbProvider.AttributeMap.FamilyName,
				GivenName:         dbProvider.AttributeMap.GivenName,
				MiddleName:        dbProvider.AttributeMap.MiddleName,
				NickName:          dbProvider.AttributeMap.NickName,
				PreferredUsername: dbProvider.AttributeMap.PreferredUsername,
				Profile:           dbProvider.AttributeMap.Profile,
				Picture:           dbProvider.AttributeMap.Picture,
				Website:           dbProvider.AttributeMap.Website,
				Gender:            dbProvider.AttributeMap.Gender,
				Birthdate:         dbProvider.AttributeMap.Birthdate,
				ZoneInfo:          dbProvider.AttributeMap.ZoneInfo,
				Locale:            dbProvider.AttributeMap.Locale,
				UpdatedAt:         dbProvider.AttributeMap.SamlUpdatedAt,
				Email:             dbProvider.AttributeMap.Email,
				EmailVerified:     dbProvider.AttributeMap.EmailVerified,
				Phone:             dbProvider.AttributeMap.Phone,
				PhoneVerified:     dbProvider.AttributeMap.PhoneVerified,
			}

			mappedProvider := samlConfig.IdentityProvider{
				Enabled:               dbProvider.Enabled,
				Name:                  dbProvider.Name,
				Domain:                dbProvider.Domain,
				MetadataUrl:           dbProvider.MetadataUrl,
				SkipEmailVerification: dbProvider.SkipEmailVerification,
				AttributeMap:          attributeMap,
			}

			sp, err := initializeServiceProvider(mappedProvider, handler.config, handler.persister)
			if err != nil {
				ctx.Logger().Error(err)
				return nil, err
			}

			serviceProviders = append(serviceProviders, *sp)
		}
	}

	return serviceProviders, nil
}
