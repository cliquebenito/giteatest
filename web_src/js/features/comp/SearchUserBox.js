import $ from 'jquery';
import {htmlEscape} from 'escape-goat';

const {appSubUrl} = window.config;
const looksLikeEmailAddressCheck = /^\S+@\S+$/;

export function initCompSearchUserBox() {
  const {orgId} = window.config.pageData;
  const $addUserForm = $('#add-member-form');
  const $submitButton = $addUserForm.find('.search-user-box__button');
  const $searchUserBox = $('#search-user-box');
  const allowEmailInput = $searchUserBox.attr('data-allow-email') === 'true';
  const allowEmailDescription = $searchUserBox.attr('data-allow-email-description');

  $searchUserBox.find('input[name=uname]').on('input', (event) => {
    const val = event.target.value;
    if (val.length === 0) {
      $submitButton.prop('disabled', true);
    }
  })

  $searchUserBox.search({
    cache: false,
    apiSettings: {
      url: `${appSubUrl}/user/search?active=1&q={query}&uid=${orgId}`,
      onResponse(response) {
        const items = [];
        const searchQuery = $searchUserBox.find('input').val();
        const searchQueryUppercase = searchQuery.toUpperCase();
        $.each(response.data, (_i, item) => {
          let title = item.login;
          if (item.full_name && item.full_name.length > 0) {
            title += ` (${htmlEscape(item.full_name)})`;
          }
          const resultItem = {
            title,
            image: item.avatar_url
          };
          if (searchQueryUppercase === item.login.toUpperCase()) {
            items.unshift(resultItem);
          } else {
            items.push(resultItem);
          }
        });

        if (allowEmailInput && items.length === 0 && looksLikeEmailAddressCheck.test(searchQuery)) {
          const resultItem = {
            title: searchQuery,
            description: allowEmailDescription
          };
          items.push(resultItem);
        }

        if (!items.length) {
          $submitButton.prop('disabled', true);
        } else {
          $submitButton.prop('disabled', false);
        }
        return {results: items};
      }
    },
    searchFields: ['login', 'full_name'],
    showNoResults: false
  });
}
