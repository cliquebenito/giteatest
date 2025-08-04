
export const RATING_MAP = {
  '1': 'A',
  '2': 'B',
  '3': 'C',
  '4': 'D',
  '5': 'E'
};


export const convertToPercent = (value) => {
  try {
    const normalized = normalizedString(value);
    const percent = Number(normalized);
    if (isNaN(percent)) {
      throw new Error(`Can't convert value: ${value} to percent.`)
    }
    return `${percent}%`
  } catch (err) {
    console.warn(err.message);
    return value;
  }
};

export const convertToRating = (value) => {
  try {
    const normalized = normalizedString(value);
    const rating = RATING_MAP[normalized];
    if (!rating) {
      throw new Error(`Can't convert value: ${value} to rating`);
    }
    return rating;
  } catch (err) {
    console.warn(err.message);
    return value;
  }
};

export const convertToInt = (value) => {
  try {
    const normalized = normalizedString(value);
    const int = Number(normalized);
    if (isNaN(int)) {
      throw new Error(`Can't convert value: ${value} to integer.`)
    }
    return int;
  } catch (err) {
    console.warn(err.message);
    return value;
  }
};

export const convertToFloat = (value, fraction = 2) => {
  try {
    const normalized = normalizedString(value);
    const float = Number(normalized).toFixed(fraction);
    if (isNaN(Number(float))) {
      throw new Error(`Can't convert value: ${value} to float.`)
    }
    return float;
  } catch (err) {
    console.warn(err.message);
    return value;
  }
};


export const convertToDate = (value) => {
  try {
    const normalized = Number(normalizedString(value));
    const date = new Date(normalized);
    if (date instanceof Date && !isNaN(date)) {
      return date.toLocaleDateString()
    } else {
      throw new Error(`Can't convert value: ${value} to date.`)
    }
  } catch (err) {
    console.warn(err.message);
    return value;
  }
}

export const convertToBool = (value) => {
  try {
    const boolSting = normalizedString(value);
    if (boolSting === 'true') {
      return 'Yes'
    } else if (boolSting === 'false') {
      return 'No'
    } else {
      throw new Error(`Can't convert value: ${value} to boolean.`)
    }
  } catch (err) {
    console.warn(err.message);
    return value;
  }
}


export const normalizedString = (value) => {
  try {
    return value.toLowerCase().trim();
  } catch (err) {
    console.warn(`Can't normalize value: ${value}`, err.message)
    return value;
  }
}
