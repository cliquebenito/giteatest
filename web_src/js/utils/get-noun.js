/**
 * Склоняет существительное с числом.
 * @param number - Число.
 * @param titles - Массив: "0": Именительный падеж единственного числа, "1": Родительный падеж единственного числа, "2": Родительный падеж множественного числа.
 */

export const getNoun = (number, titles) => {
  const cases = [2, 0, 1, 1, 1, 2];
  return titles[
    number % 100 > 4 && number % 100 < 20
      ? 2
      : cases[number % 10 < 5 ? number % 10 : 5]
  ];
};
