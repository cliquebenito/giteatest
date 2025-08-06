// https://stackoverflow.com/a/70342741

function makeHash(url, obj) {
  // put properties in sorted order to make the hash canonical
  // the canonical sort is top level only,
  //    does not sort properties in nested objects
  const items = Object.entries(obj).sort((a, b) => b[0].localeCompare(a[0]));
  // add URL on the front
  items.unshift(url);
  return JSON.stringify(items);
}

async function globalFetch(resource, init = {}) {
  const key = makeHash(resource, init);

  const now = Date.now();
  const expirationDuration = 5 * 1000;
  const newExpiration = now + expirationDuration;

  const cachedItem = globalFetch.cache.get(key);
  // if we found an item and it expires in the future (not expired yet)
  if (cachedItem && cachedItem.expires >= now) {
    // update expiration time
    cachedItem.expires = newExpiration;
    return cachedItem.promise;
  }

  // couldn't use a value from the cache
  // make the request
  const p = fetch(resource, init);
  p.then((response) => {
    if (!response.ok) {
      // if response not OK, remove it from the cache
      globalFetch.cache.delete(key);
    }
  }, () => {
    // if promise rejected, remove it from the cache
    globalFetch.cache.delete(key);
  });
  // save this promise (will replace any expired value already in the cache)
  globalFetch.cache.set(key, {promise: p, expires: newExpiration});
  return p;
}
// initalize cache
globalFetch.cache = new Map();

// clean up interval timer to remove expired entries
// does not need to run that often because .expires is already checked above
// this just cleans out old expired entries to avoid memory increasing
// indefinitely
globalFetch.interval = setInterval(() => {
  const now = Date.now();
  for (const [key, value] of globalFetch.cache) {
    if (value.expires < now) {
      globalFetch.cache.delete(key);
    }
  }
}, 10 * 60 * 1000); // run every 10 minutes


window.globalFetch = globalFetch;
