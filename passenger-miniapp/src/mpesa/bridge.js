/**
 * M-Pesa Ethiopia HTML5 Mini Program helpers (hylid-bridge).
 * Platform features required: PaymentET + UserScopes.
 * Pay: call('payWithMpesa', …)  |  User: call('userScopes', …)
 */

export function isMiniProgram() {
  return /MiniProgram/i.test(navigator.userAgent || '')
}

function myApi() {
  return typeof window !== 'undefined' ? window.my : null
}

/** Ethiopia SuperApp operations via hylid-bridge `call`. */
export function bridgeCall(operation, options = {}) {
  const api = myApi()
  const callFn = api?.call
  if (typeof callFn !== 'function') {
    return Promise.reject(new Error(`${operation} unavailable — open inside M-Pesa Mini Program`))
  }
  return new Promise((resolve, reject) => {
    callFn(operation, {
      ...options,
      success: (res) => {
        if (res?.error != null && res.error !== 0) {
          reject(Object.assign(new Error(res.errorMessage || `Error ${res.error}`), res))
          return
        }
        resolve(res)
      },
      fail: (err) => {
        const msg = err?.errorMessage || err?.message || `${operation} failed`
        reject(Object.assign(new Error(msg), err || {}))
      },
    })
  })
}

/** Promisify classic `my.*` JSAPIs (alert, toast, nav, …). */
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

const DEFAULT_SCOPES = [
  'USER_NAME',
  'USER_LNAME',
  'USER_MNAME',
  'USER_MOBILE',
  'USER_ID',
]

/**
 * Ethiopia SSO / KYC — call('userScopes').
 * Returns { passenger_name, passenger_phone, user_id, kycAllowed, raw }.
 */
export async function getUserScopes(scopes = DEFAULT_SCOPES) {
  if (!isMiniProgram()) {
    return { mock: true, passenger_name: '', passenger_phone: '', user_id: null, kycAllowed: [] }
  }
  const res = await bridgeCall('userScopes', { scopes })
  const map = {}
  for (const item of res?.kycAllowed || []) {
    if (item?.key) map[item.key] = item.value
  }
  const first = map.USER_NAME || ''
  const mid = map.USER_MNAME || ''
  const last = map.USER_LNAME || ''
  const name = [first, mid, last].filter(Boolean).join(' ').trim()
  let phone = String(map.USER_MOBILE || '').replace(/\s+/g, '')
  if (phone.startsWith('251') && phone.length === 12) phone = `0${phone.slice(3)}`
  return {
    mock: false,
    passenger_name: name,
    passenger_phone: phone,
    user_id: map.USER_ID || null,
    kycAllowed: res?.kycAllowed || [],
    kycRejected: res?.kycRejected || [],
    raw: res,
  }
}

/**
 * Ethiopia Pay with M-Pesa — call('payWithMpesa').
 * businessID = merchant shortcode / till.
 * Success: { transactionId }
 */
export function payWithMpesa({
  businessID,
  amount,
  billReference,
  reason = 'GuzoPay fare',
  currency = 'ETB',
} = {}) {
  if (!isMiniProgram()) {
    return Promise.reject(new Error('payWithMpesa only works inside M-Pesa Mini Program'))
  }
  if (!businessID) {
    return Promise.reject(new Error('VITE_MPESA_BUSINESS_ID / businessID required'))
  }
  return bridgeCall('payWithMpesa', {
    businessID: String(businessID),
    billReference: billReference ? String(billReference) : undefined,
    amount: typeof amount === 'number' ? amount.toFixed(1) : String(amount),
    currency,
    reason,
  })
}
