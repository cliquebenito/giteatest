function fetchMock(data, delay, err) {
    return (url, options) => {
        new Promise((resolve, reject) => {
            console.log('run request');
            console.log(url, options);
            if (err) {
                setTimeout(() => {
                    reject({
                        ok: false,
                        status: 400,
                        json: () => Promise.resolve(data),
                    })
                }, delay);
            } else {
                setTimeout(() => {
                    resolve({
                        ok: true,
                        status: 200,
                        json: () => Promise.resolve(data),
                    })
                }, delay);
            }
        });
    }
}