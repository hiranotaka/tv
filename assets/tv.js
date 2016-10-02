function updatePositions() {
    $('td.hour').css('left', $(window).scrollLeft() + 'px');
    $('td.program').css('top', $(window).scrollTop() + 'px');
}
$(window).scroll(function () {
    updatePositions();
});
updatePositions();
