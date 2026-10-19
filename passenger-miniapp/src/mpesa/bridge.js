/**
 * M-Pesa SuperApp HTML5 Mini Program JSBridge helpers.
 * Docs: https://developer.safaricom.et/mini-app-documentation
 * Bridge SDK: https://cdn.marmot-cloud.com/npm/hylid-bridge/2.10.0/index.js
 */

export function isMiniProgram() {
  return /MiniProgram/i.test(navigator.userAgent || '')
}

function myApi() {
  if (typeof window !== 'undefined' && window.my) return window.my
  return null
}

/** Promisify a `my.*` JSAPI call. */
export function callMy(method, options = {}) {
  const api = myApi()
  if (!api || typeof api[method] !== 'function') {
    return Promise.reject(new Error(`${method} unavailable outside M-Pesa Mini Program`))
  }
  return new Promise((resolve, reject) => {
    api[method]({
      ...options,
      success: (res) => resolve(res),
      fail: (err) => reject(err || new Error(`${method} failed`)),
    })
  })
}

export function setNavigationBar(title = 'GuzoPay') {
  if (!isMiniProgram()) return Promise.resolve()
  return callMy('setNavigationBar', { title }).catch(() => {})
}

export function showLoading(content = 'Loading…') {
  if (!isMiniProgram()) return Promise.resolve()
  return callMy('showLoading', { content }).catch(() => {})
}

export function hideLoading() {
  if (!isMiniProgram()) return Promise.resolve()
  return callMy('hideLoading').catch(() => {})
}

export function showToast(content, duration = 2000) {
  if (!isMiniProgram()) return Promise.resolve()
  return callMy('showToast', { content, duration }).catch(() => {})
}

export function alert({ title = 'GuzoPay', content, buttonText = 'OK' } = {}) {
  if (!isMiniProgram()) {
    window.alert(content || title)
    return Promise.resolve()
  }
  return callMy('alert', { title, content, buttonText })
}

/** Silent/user auth — returns { authCode, authSuccessScopes }. */
export function getAuthCode(scopes = ['auth_user']) {
  if (!isMiniProgram()) {
    return Promise.resolve({
      authCode: null,
      authSuccessScopes: [],
      mock: true,
    })
  }
  return callMy('getAuthCode', { scopes })
}

/**
 * Trigger M-Pesa wallet cashier.
 * resultCode: 9000 success, 8000 processing, 4000 fail, 6001 cancel, 6002 network
 */
export function tradePay({ tradeNO, orderStr, paymentUrl } = {}) {
  if (!isMiniProgram()) {
    return Promise.reject(new Error('tradePay only works inside M-Pesa Mini Program'))
  }
  const payload = {}
  if (tradeNO) payload.tradeNO = tradeNO
  if (orderStr) payload.orderStr = orderStr
  if (paymentUrl) payload.paymentUrl = paymentUrl
  return callMy('tradePay', payload)
}

export function getLocation() {
  if (!isMiniProgram()) return Promise.resolve(null)
  return callMy('getLocation', { type: 1 }).catch(() => null)
}
