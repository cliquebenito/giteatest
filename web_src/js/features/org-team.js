import $ from 'jquery';
import {hideElem, showElem} from '../utils/dom.js';

const {appSubUrl} = window.config;

export function initOrgTeamSettings() {

  const $createNewOrgTeamForm = $('#create-new-org-team');
  const $teamNameInput = $('input#team_name');
  const $teamNameField = $teamNameInput.parent('.field');
  const $errorMessage = $teamNameField.find('.error-message');
  const RESERVED_TEAM_NAMES = ['new'];

  $createNewOrgTeamForm.on('submit', () => {
    let message = 'Can\'t create team with resevred name';
    if (window.config.lang === 'ru-RU') {
      message = 'Нельзя использовать в названии команды зарезервированное слово';
    }

    const teamName = $teamNameInput.val() || '';
    if (RESERVED_TEAM_NAMES.includes(teamName.toLowerCase())) {
      $errorMessage.removeClass('hidden').text(`${message}: ${teamName}`);
      $teamNameField.addClass('error').get(0).scrollIntoView({ behavior: 'smooth' });
      return false;
    }
    $teamNameField.removeClass('error');
    $errorMessage.addClass('hidden').text('');

  });


  // Change team access mode
  $('.organization.new.team input[name=permission]').on('change', () => {
    const val = $('input[name=permission]:checked', '.organization.new.team').val();
    if (val === 'admin') {
      hideElem($('.organization.new.team .team-units'));
    } else {
      showElem($('.organization.new.team .team-units'));
    }
  });
}


export function initOrgTeamSearchRepoBox() {
  const $searchRepoBox = $('#search-repo-box');
  $searchRepoBox.search({
    minCharacters: 2,
    apiSettings: {
      url: `${appSubUrl}/repo/search?q={query}&uid=${$searchRepoBox.data('uid')}`,
      onResponse(response) {
        const items = [];
        $.each(response.data, (_i, item) => {
          items.push({
            title: item.repository.full_name.split('/')[1],
            description: item.repository.full_name
          });
        });

        return {results: items};
      }
    },
    searchFields: ['full_name'],
    showNoResults: false
  });
}
