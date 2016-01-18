import React from 'react';

import Service from './service';

export default class ServicesList extends React.Component {

  render() {
    const services = this.props.services.map(service =>
      <Service {...service} />
    );
    return (
      <div className="service-list">
        {services}
      </div>
    );
  }
}
