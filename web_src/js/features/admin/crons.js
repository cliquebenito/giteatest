export function initCronSettings() {
  const $cronTableChevrons = $('.cron-table__cell_name button');
  $cronTableChevrons.click(function (event) {
    event.preventDefault();
    $(this).toggleClass('active')
    const $alertRow = $(this).parents('tr').next('.cron-table__row_alert');
    $alertRow.toggleClass('active')
  });
}
