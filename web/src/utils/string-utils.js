export function labels(kv) {
  const s = [];
  Object.keys(kv).forEach(k => {
    s.push(k + '=' + kv[k]);
  });
  return s.join(', ');
}

export function maybeTruncate(id) {
  if (id.length > 12) {
    return id.substr(0, 12) + '...';
  }
  return id;
}

export function plural(text, count) {
  if (count === 1) {
    return text;
  }
  return `${text}s`;
}
